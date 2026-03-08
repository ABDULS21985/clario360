# SC-005: Capacity Planning Guide

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | SC-005                                     |
| **Title**        | Capacity Planning Guide                    |
| **Category**     | Scaling                                    |
| **Severity**     | Low (Proactive)                            |
| **Author**       | Platform Engineering                       |
| **Created**      | 2026-03-08                                 |
| **Last Updated** | 2026-03-08                                 |
| **Review Cycle** | Monthly                                    |
| **Platform**     | GCP (GKE)                                  |
| **Namespace**    | clario360                                  |

---

## Summary

This runbook provides a comprehensive capacity planning guide for the Clario 360 platform. It includes procedures for generating current resource utilization reports, growth projections based on tenant count, per-service resource requirements, storage growth estimates, network bandwidth requirements, cost optimization recommendations, and a scaling decision matrix. Use this runbook proactively on a monthly basis or when onboarding a significant number of new tenants.

---

## Prerequisites

- `kubectl` CLI configured with cluster credentials
- `gcloud` CLI authenticated with billing viewer access
- Access to Prometheus/Grafana for historical metrics
- `psql` client for database sizing queries
- Access to the `clario360`, `kafka`, and `kube-system` namespaces

### Verify Access

```bash
gcloud container clusters get-credentials clario360-cluster \
  --region us-central1 \
  --project clario360-prod

kubectl get namespaces
```

---

## Procedure

### Step 1: Current Resource Utilization Report

#### 1a: Cluster-Level CPU and Memory

```bash
# Overall cluster resource utilization
kubectl top nodes

# Detailed node capacity vs allocated vs actual usage
kubectl get nodes -o json | kubectl get nodes -o custom-columns=\
"NAME:.metadata.name,\
CPU_CAPACITY:.status.capacity.cpu,\
MEM_CAPACITY:.status.capacity.memory,\
CPU_ALLOCATABLE:.status.allocatable.cpu,\
MEM_ALLOCATABLE:.status.allocatable.memory"

# Per-node resource allocation percentage
kubectl describe nodes | grep -A 6 "Allocated resources:"

# Total cluster capacity summary
echo "=== Cluster Resource Summary ==="
echo "Nodes: $(kubectl get nodes --no-headers | wc -l)"
echo "Total CPU Capacity: $(kubectl get nodes -o jsonpath='{range .items[*]}{.status.capacity.cpu}{"\n"}{end}' | paste -sd+ | bc) cores"
echo "Total Memory Capacity: $(kubectl get nodes -o jsonpath='{range .items[*]}{.status.capacity.memory}{"\n"}{end}' | head -1) (per node)"
```

#### 1b: Namespace-Level Resource Usage

```bash
# Resource usage by namespace
kubectl top pods -n clario360 --sort-by=cpu
kubectl top pods -n clario360 --sort-by=memory

# Resource requests vs limits for all clario360 pods
kubectl get pods -n clario360 -o json | \
  kubectl get pods -n clario360 -o custom-columns=\
"NAME:.metadata.name,\
CPU_REQ:.spec.containers[0].resources.requests.cpu,\
CPU_LIM:.spec.containers[0].resources.limits.cpu,\
MEM_REQ:.spec.containers[0].resources.requests.memory,\
MEM_LIM:.spec.containers[0].resources.limits.memory"

# Resource quota usage (if quotas are configured)
kubectl get resourcequota -n clario360 -o yaml
```

#### 1c: Disk Utilization

```bash
# PVC usage across all namespaces
kubectl get pvc --all-namespaces -o custom-columns=\
"NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
STATUS:.status.phase,\
CAPACITY:.status.capacity.storage,\
STORAGECLASS:.spec.storageClassName"

# Check actual disk usage on PostgreSQL
kubectl exec -n clario360 postgresql-0 -- df -h /var/lib/postgresql/data

# Check actual disk usage on Kafka brokers
for i in 0 1 2; do
  echo "=== kafka-${i} ==="
  kubectl exec -n kafka kafka-${i} -- df -h /var/lib/kafka/data
done

# Check Redis memory usage
kubectl exec -n clario360 redis-0 -- redis-cli -h redis INFO memory | grep -E "used_memory_human|maxmemory_human|mem_fragmentation_ratio"
```

#### 1d: Network Utilization

```bash
# Check service traffic patterns (via Prometheus if available)
# PromQL queries to run in Grafana:
# - sum(rate(container_network_receive_bytes_total{namespace="clario360"}[5m])) by (pod)
# - sum(rate(container_network_transmit_bytes_total{namespace="clario360"}[5m])) by (pod)

# Check ingress traffic
kubectl get svc -n clario360 -o wide

# Check current network policies
kubectl get networkpolicies -n clario360
```

### Step 2: Growth Projections Based on Tenant Count

#### Current Tenant Metrics

```bash
# Get current tenant count
kubectl port-forward svc/postgresql -n clario360 5432:5432 &
export PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 \
  -o jsonpath='{.data.postgres-password}' | base64 -d)

psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT count(*) AS total_tenants,
       count(*) FILTER (WHERE status = 'active') AS active_tenants,
       count(*) FILTER (WHERE created_at > now() - interval '30 days') AS new_last_30d
FROM tenants;
"

# Get per-tenant user counts
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT t.name AS tenant,
       count(u.id) AS user_count
FROM tenants t
LEFT JOIN users u ON u.tenant_id = t.id
WHERE t.status = 'active'
GROUP BY t.name
ORDER BY user_count DESC
LIMIT 20;
"
```

#### Growth Projection Table

Use the following table to project resource needs based on tenant growth. Values are based on observed per-tenant resource consumption averages.

| Metric                    | Per Tenant (avg) | 50 Tenants | 100 Tenants | 250 Tenants | 500 Tenants |
|---------------------------|-----------------|------------|-------------|-------------|-------------|
| **Users**                 | 25              | 1,250      | 2,500       | 6,250       | 12,500      |
| **API Requests/min**      | 200             | 10,000     | 20,000      | 50,000      | 100,000     |
| **Concurrent WebSockets** | 15              | 750        | 1,500       | 3,750       | 7,500       |
| **Audit Events/day**      | 5,000           | 250,000    | 500,000     | 1,250,000   | 2,500,000   |
| **DB Storage (GB)**       | 2               | 100        | 200         | 500         | 1,000       |
| **Kafka Messages/day**    | 10,000          | 500,000    | 1,000,000   | 2,500,000   | 5,000,000   |
| **File Storage (GB)**     | 5               | 250        | 500         | 1,250       | 2,500       |
| **API Gateway Pods**      | -               | 4          | 8           | 14          | 20          |
| **Total vCPU**            | -               | 32         | 64          | 128         | 256         |
| **Total Memory (GB)**     | -               | 128        | 256         | 512         | 1,024       |
| **GKE Nodes (e2-std-4)**  | -               | 8          | 16          | 32          | 64          |
| **Monthly Cost (est.)**   | -               | $3,200     | $6,000      | $14,000     | $28,000     |

### Step 3: Per-Service Resource Requirements

| Service              | CPU Request | CPU Limit | Memory Request | Memory Limit | Replicas (50T) | Replicas (100T) | Replicas (250T) | Replicas (500T) |
|----------------------|------------|-----------|----------------|-------------|-----------------|-----------------|-----------------|-----------------|
| api-gateway          | 500m       | 1000m     | 512Mi          | 1Gi         | 4               | 8               | 14              | 20              |
| iam-service          | 250m       | 500m      | 256Mi          | 512Mi       | 2               | 4               | 6               | 10              |
| audit-service        | 250m       | 500m      | 256Mi          | 512Mi       | 2               | 4               | 6               | 8               |
| workflow-engine      | 500m       | 1000m     | 512Mi          | 1Gi         | 2               | 4               | 8               | 12              |
| notification-service | 250m       | 500m      | 256Mi          | 512Mi       | 2               | 4               | 6               | 10              |
| cyber-service        | 500m       | 1000m     | 512Mi          | 1Gi         | 2               | 4               | 6               | 10              |
| data-service         | 500m       | 1000m     | 1Gi            | 2Gi         | 2               | 4               | 6               | 10              |
| acta-service         | 250m       | 500m      | 512Mi          | 1Gi         | 2               | 3               | 5               | 8               |
| lex-service          | 250m       | 500m      | 256Mi          | 512Mi       | 2               | 3               | 5               | 8               |
| visus-service        | 250m       | 500m      | 512Mi          | 1Gi         | 2               | 3               | 5               | 8               |
| **PostgreSQL**       | 2000m      | 4000m     | 8Gi            | 16Gi        | 1+2R            | 1+2R            | 1+3R            | 1+4R            |
| **Kafka (per broker)**| 1000m     | 2000m     | 4Gi            | 8Gi         | 3               | 3               | 5               | 7               |
| **Redis**            | 500m       | 1000m     | 2Gi            | 4Gi         | 3 (sentinel)    | 3 (sentinel)    | 6 (cluster)     | 6 (cluster)     |
| **PgBouncer**        | 250m       | 500m      | 256Mi          | 512Mi       | 2               | 2               | 3               | 4               |

> **Note:** "R" in PostgreSQL replicas column refers to read replicas. "50T" means 50 tenants, etc.

### Step 4: Storage Growth Estimates

#### 4a: PostgreSQL Storage

```bash
# Current database sizes
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT datname,
       pg_size_pretty(pg_database_size(datname)) AS current_size,
       pg_database_size(datname) AS size_bytes
FROM pg_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY pg_database_size(datname) DESC;
"

# Table-level storage for the largest database
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT schemaname || '.' || tablename AS table_name,
       pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS total_size,
       pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS data_size,
       pg_size_pretty(pg_indexes_size(schemaname || '.' || tablename::regclass)) AS index_size,
       n_live_tup AS row_count
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
LIMIT 20;
"

# WAL disk usage
psql -h 127.0.0.1 -U postgres -d platform_core -c "
SELECT pg_size_pretty(sum(size)) AS total_wal_size
FROM pg_ls_waldir();
"
```

#### Storage Growth Projection Table

| Database       | Current (est.) | +6 months | +12 months | +24 months | Notes                              |
|----------------|---------------|-----------|------------|------------|-------------------------------------|
| platform_core  | 20 GB         | 40 GB     | 75 GB      | 140 GB     | Audit logs dominate growth          |
| cyber_db       | 15 GB         | 30 GB     | 55 GB      | 100 GB     | Scan results accumulate             |
| data_db        | 25 GB         | 50 GB     | 95 GB      | 180 GB     | Data catalog + ingestion metadata   |
| acta_db        | 10 GB         | 20 GB     | 38 GB      | 70 GB      | Document metadata (files in GCS)    |
| lex_db         | 5 GB          | 8 GB      | 14 GB      | 25 GB      | Regulatory data; slower growth      |
| visus_db       | 8 GB          | 15 GB     | 28 GB      | 50 GB      | Report definitions + cached results |
| **Total PG**   | **83 GB**     | **163 GB**| **305 GB** | **565 GB** |                                     |

#### 4b: Kafka Storage

```bash
# Check Kafka disk usage per broker
for i in 0 1 2; do
  echo "=== kafka-${i} ==="
  kubectl exec -n kafka kafka-${i} -- du -sh /var/lib/kafka/data/
  kubectl exec -n kafka kafka-${i} -- du -sh /var/lib/kafka/data/clario360.* 2>/dev/null | sort -rh | head -10
done
```

| Component         | Current (est.) | +6 months | +12 months | +24 months | Notes                              |
|-------------------|---------------|-----------|------------|------------|-------------------------------------|
| Kafka (per broker)| 50 GB         | 75 GB     | 100 GB     | 150 GB     | 7-day retention; grows with tenants |
| Kafka (total)     | 150 GB        | 225 GB    | 500 GB     | 1,050 GB   | Includes replication overhead       |

#### 4c: Log Storage

| Component            | Daily Volume | 30-Day Retention | 90-Day Retention | Notes                          |
|----------------------|-------------|------------------|------------------|--------------------------------|
| Application logs     | 5 GB        | 150 GB           | 450 GB           | All services combined           |
| Audit logs (immutable)| 2 GB       | 60 GB            | 180 GB           | Retained indefinitely in GCS   |
| Kafka logs           | 3 GB        | 90 GB            | 270 GB           | Internal Kafka operation logs  |
| GKE system logs      | 1 GB        | 30 GB            | 90 GB            | kubelet, kube-proxy, etc.      |
| **Total**            | **11 GB**   | **330 GB**       | **990 GB**       |                                |

### Step 5: Network Bandwidth Requirements

#### Bandwidth Estimation by Component

| Traffic Path                          | Avg Bandwidth | Peak Bandwidth | Protocol    | Notes                              |
|---------------------------------------|--------------|----------------|-------------|-------------------------------------|
| External -> api-gateway (ingress)     | 50 Mbps      | 200 Mbps       | HTTPS       | Client API calls                    |
| api-gateway -> backend services       | 80 Mbps      | 300 Mbps       | gRPC/HTTP   | Internal service-to-service         |
| Services -> PostgreSQL                | 30 Mbps      | 120 Mbps       | TCP         | Database queries/writes             |
| Services -> Kafka                     | 40 Mbps      | 160 Mbps       | TCP         | Event publishing                    |
| Kafka -> Consumer services            | 40 Mbps      | 160 Mbps       | TCP         | Event consumption                   |
| Kafka inter-broker replication        | 30 Mbps      | 100 Mbps       | TCP         | RF=3 means 2x replication traffic   |
| PostgreSQL -> Read replicas           | 10 Mbps      | 50 Mbps        | TCP         | WAL streaming                       |
| Services -> Redis                     | 20 Mbps      | 80 Mbps        | TCP         | Cache reads/writes                  |
| WebSocket connections (persistent)    | 10 Mbps      | 40 Mbps        | WSS         | Real-time notifications             |
| Services -> GCS (file storage)        | 20 Mbps      | 100 Mbps       | HTTPS       | Document uploads/downloads          |
| **Total Intra-Cluster**              | **330 Mbps** | **1,310 Mbps** |             |                                     |

#### Monitor Network Usage

```bash
# Check network usage via node metrics
kubectl top nodes

# Check network policies that may throttle traffic
kubectl get networkpolicies -n clario360 -o yaml

# Check GKE network tier
gcloud container clusters describe clario360-cluster \
  --region us-central1 \
  --format="get(networkConfig)"
```

### Step 6: Cost Optimization Recommendations

#### 6a: Right-Size Resource Requests

```bash
# Identify over-provisioned pods (CPU request > 2x actual usage)
kubectl top pods -n clario360 --no-headers | while read pod cpu mem; do
  CPU_USAGE=$(echo ${cpu} | sed 's/m//')
  CPU_REQUEST=$(kubectl get pod ${pod} -n clario360 -o jsonpath='{.spec.containers[0].resources.requests.cpu}' 2>/dev/null | sed 's/m//')
  if [ -n "${CPU_REQUEST}" ] && [ "${CPU_REQUEST}" -gt 0 ] 2>/dev/null; then
    RATIO=$((CPU_REQUEST / (CPU_USAGE + 1)))
    if [ "${RATIO}" -gt 2 ]; then
      echo "OVER-PROVISIONED: ${pod} - Using ${cpu} / Requested ${CPU_REQUEST}m (${RATIO}x over)"
    fi
  fi
done

# Identify under-provisioned pods (CPU usage > 80% of request)
kubectl top pods -n clario360 --no-headers | while read pod cpu mem; do
  CPU_USAGE=$(echo ${cpu} | sed 's/m//')
  CPU_REQUEST=$(kubectl get pod ${pod} -n clario360 -o jsonpath='{.spec.containers[0].resources.requests.cpu}' 2>/dev/null | sed 's/m//')
  if [ -n "${CPU_REQUEST}" ] && [ "${CPU_REQUEST}" -gt 0 ] 2>/dev/null; then
    USAGE_PCT=$((CPU_USAGE * 100 / CPU_REQUEST))
    if [ "${USAGE_PCT}" -gt 80 ]; then
      echo "UNDER-PROVISIONED: ${pod} - Using ${cpu} / Requested ${CPU_REQUEST}m (${USAGE_PCT}%)"
    fi
  fi
done
```

#### 6b: Cost Optimization Checklist

| Optimization                         | Estimated Savings | Effort | Risk   | Details                                          |
|--------------------------------------|------------------|--------|--------|--------------------------------------------------|
| Right-size resource requests         | 15-25%           | Low    | Low    | Match requests to P95 actual usage               |
| Use Spot VMs for non-critical pools  | 60-70% per node  | Medium | Medium | Batch jobs, visus-service report generation      |
| Enable committed use discounts (CUD) | 20-35%           | Low    | Low    | 1-year or 3-year commitments for base capacity   |
| Use E2 shared-core for dev/staging   | 40-50%           | Low    | Low    | e2-medium instead of e2-standard-4               |
| Consolidate low-usage services       | 10-15%           | High   | Medium | Co-locate lex-service and acta-service           |
| Implement autoscaling (HPA + CA)     | 20-30%           | Medium | Low    | Scale down during off-hours                      |
| Optimize PG storage (archive old data)| 10-20%          | Medium | Low    | Move audit logs > 1yr to cold storage (GCS)      |
| Reduce Kafka retention period        | 15-25%           | Low    | Low    | 7 days -> 3 days for non-critical topics         |
| Use regional PD instead of SSD       | 30-40% on disk   | Low    | Low    | For non-latency-sensitive storage                |

#### 6c: Check Current GCP Billing

```bash
# List current GKE billing (requires Billing Viewer role)
gcloud billing accounts list

# Export billing data for the project
gcloud beta billing budgets list --billing-account=BILLING_ACCOUNT_ID

# Check committed use discounts
gcloud compute commitments list --project clario360-prod

# Check sustained use discounts (automatic)
gcloud compute instances list --project clario360-prod \
  --format="table(name, zone, machineType, status, scheduling.preemptible)"
```

### Step 7: Scaling Decision Matrix

Use this matrix to determine what to scale based on the observed symptom.

| Symptom                                           | Primary Action                    | Secondary Action               | Runbook         |
|---------------------------------------------------|-----------------------------------|--------------------------------|-----------------|
| API latency > 500ms (P95)                         | Scale api-gateway pods (HPA)     | Add GKE nodes if pending       | SC-001, SC-004  |
| Database connection errors / timeouts             | Add PgBouncer capacity           | Increase max_connections       | SC-002          |
| Database query latency > 200ms                    | Add read replicas                | Vertical scale PG              | SC-002          |
| Kafka consumer lag > 10,000 messages              | Scale consumer service pods      | Increase topic partitions      | SC-001, SC-003  |
| Kafka broker disk > 80%                           | Reduce retention / add brokers   | Increase broker PVC            | SC-003          |
| Pods in Pending state (Insufficient CPU)          | Scale node pool / add nodes      | Enable cluster autoscaler      | SC-004          |
| Pods in Pending state (Insufficient Memory)       | Add high-memory node pool        | Right-size pod memory limits   | SC-004          |
| WebSocket disconnections / connection refused      | Scale notification-service       | Scale api-gateway              | SC-001          |
| Redis memory > 80% maxmemory                      | Scale Redis cluster              | Review eviction policy         | --              |
| PostgreSQL replication lag > 10s                   | Increase replica resources       | Reduce write-heavy queries     | SC-002          |
| Workflow execution backlog growing                 | Scale workflow-engine            | Increase Kafka partitions      | SC-001, SC-003  |
| File upload timeouts                              | Scale acta-service               | Increase GCS throughput        | SC-001          |
| Report generation > 30s                           | Scale visus-service              | Add compute node pool          | SC-001, SC-004  |
| Tenant onboarding > 5 tenants/week                | Proactive capacity increase      | Review growth projections      | SC-005          |
| Monthly cost > 120% of budget                     | Right-size resources             | Review optimization checklist  | SC-005          |

#### Scaling Thresholds Quick Reference

| Metric                               | Warning Threshold | Critical Threshold | Action Required              |
|--------------------------------------|------------------|--------------------|------------------------------|
| Node CPU utilization                 | > 70%            | > 85%              | Add nodes or enable CA       |
| Node memory utilization              | > 75%            | > 90%              | Add nodes or resize          |
| Pod CPU utilization (avg)            | > 65%            | > 80%              | HPA scale-up trigger         |
| Pod memory utilization (avg)         | > 70%            | > 85%              | HPA scale-up trigger         |
| PostgreSQL connections               | > 70% of max     | > 85% of max       | PgBouncer / increase max     |
| PostgreSQL replication lag           | > 5s             | > 30s              | Investigate / scale replica  |
| PostgreSQL disk usage                | > 70%            | > 85%              | Expand PVC / archive data    |
| Kafka consumer lag (messages)        | > 5,000          | > 50,000           | Scale consumers / partitions |
| Kafka broker disk usage              | > 70%            | > 85%              | Add brokers / reduce retention|
| Redis memory usage                   | > 70%            | > 85%              | Scale Redis cluster          |
| API response time (P95)              | > 300ms          | > 1000ms           | Scale api-gateway            |
| Pending pods                         | > 0 for 2min     | > 0 for 5min       | Scale node pool              |

### Step 8: Generate Monthly Capacity Report

Run this script monthly to generate a capacity snapshot.

```bash
#!/bin/bash
# Capacity Report Generator for Clario 360
# Run: bash capacity-report.sh > capacity-report-$(date +%Y-%m).txt

echo "============================================="
echo "  Clario 360 Capacity Report"
echo "  Generated: $(date)"
echo "============================================="

echo ""
echo "=== CLUSTER OVERVIEW ==="
echo "Nodes:"
kubectl get nodes --no-headers | wc -l
echo ""
kubectl top nodes

echo ""
echo "=== NODE POOLS ==="
gcloud container node-pools list \
  --cluster clario360-cluster \
  --region us-central1 \
  --format="table(name, config.machineType, autoscaling.minNodeCount, autoscaling.maxNodeCount, initialNodeCount)"

echo ""
echo "=== NAMESPACE: clario360 ==="
echo "Deployments:"
kubectl get deployments -n clario360 --no-headers | awk '{print $1, "Replicas:", $2}'

echo ""
echo "Pod Resource Usage:"
kubectl top pods -n clario360 --sort-by=cpu

echo ""
echo "=== HPA STATUS ==="
kubectl get hpa -n clario360

echo ""
echo "=== PVC USAGE ==="
kubectl get pvc -n clario360 -o custom-columns="NAME:.metadata.name,CAPACITY:.status.capacity.storage,STATUS:.status.phase"
kubectl get pvc -n kafka -o custom-columns="NAME:.metadata.name,CAPACITY:.status.capacity.storage,STATUS:.status.phase"

echo ""
echo "=== DATABASE SIZES ==="
export PGPASSWORD=$(kubectl get secret postgresql-credentials -n clario360 \
  -o jsonpath='{.data.postgres-password}' | base64 -d)
kubectl port-forward svc/postgresql -n clario360 5432:5432 &
PF_PID=$!
sleep 3
psql -h 127.0.0.1 -U postgres -d platform_core -t -c "
SELECT datname || ': ' || pg_size_pretty(pg_database_size(datname))
FROM pg_database
WHERE datname IN ('platform_core','cyber_db','data_db','acta_db','lex_db','visus_db')
ORDER BY pg_database_size(datname) DESC;
"
kill ${PF_PID} 2>/dev/null

echo ""
echo "=== KAFKA CLUSTER ==="
echo "Brokers: $(kubectl get pods -n kafka -l app=kafka --no-headers | wc -l)"
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe 2>/dev/null | grep -c "Topic:"
echo " topics total"

echo ""
echo "=== REDIS ==="
kubectl exec -n clario360 redis-0 -- redis-cli -h redis INFO memory 2>/dev/null | grep -E "used_memory_human|maxmemory_human"

echo ""
echo "=== PENDING PODS ==="
kubectl get pods --all-namespaces --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l
echo " pending pods"

echo ""
echo "============================================="
echo "  End of Report"
echo "============================================="
```

---

## Verification

After completing capacity planning analysis, confirm:

1. The capacity report was generated successfully:
   ```bash
   ls -la capacity-report-$(date +%Y-%m).txt
   ```

2. All monitoring dashboards are accessible and showing data:
   ```bash
   kubectl get svc -n monitoring
   ```

3. Alerts are configured for all critical thresholds:
   ```bash
   kubectl get prometheusrules -n monitoring
   ```

4. Growth projections have been reviewed and shared with stakeholders.

5. Cost optimization recommendations have been evaluated and applicable ones scheduled.

---

## Related Links

| Document                  | Link                                             |
|---------------------------|--------------------------------------------------|
| Horizontal Pod Scaling    | [SC-001-horizontal-scaling.md](SC-001-horizontal-scaling.md) |
| Database Scaling          | [SC-002-database-scaling.md](SC-002-database-scaling.md)     |
| Kafka Scaling             | [SC-003-kafka-scaling.md](SC-003-kafka-scaling.md)           |
| Node Pool Scaling         | [SC-004-node-pool-scaling.md](SC-004-node-pool-scaling.md)   |
| GKE Pricing Calculator    | https://cloud.google.com/products/calculator                 |
| GCP Committed Use Discounts | https://cloud.google.com/compute/docs/instances/committed-use-discounts-overview |
