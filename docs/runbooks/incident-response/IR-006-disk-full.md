# IR-006: Disk Space Exhaustion

| Field            | Value                                    |
|------------------|------------------------------------------|
| Runbook ID       | IR-006                                   |
| Title            | Disk Space Exhaustion                    |
| Severity         | P2 — High                               |
| Owner            | Platform Team                            |
| Last Updated     | 2026-03-08                               |
| Review Frequency | Quarterly                                |
| Approver         | Platform Lead                            |

---

## Summary

This runbook covers diagnosis and resolution of disk space exhaustion across the Clario 360 platform. Disk space issues manifest as PersistentVolumeClaim (PVC) capacity exhaustion, node-level disk pressure, or log volume saturation. Left unresolved, disk exhaustion causes pod evictions, database write failures, and service outages.

---

## Symptoms

- **Alerts**: `KubePersistentVolumeFillingUp`, `NodeDiskPressure`, `KubeletTooManyPods`
- Pods entering `Evicted` or `CrashLoopBackOff` status
- Database write errors: `FATAL: could not write to file` or `No space left on device`
- Kafka brokers refusing new messages: `kafka.common.errors.LogDirNotFoundException`
- Application logs stop writing; services return HTTP 500 on write operations
- `kubectl describe node` shows `DiskPressure` condition as `True`
- Grafana dashboard `Cluster Overview` (`/d/cluster-overview`) shows disk utilization above 85%

---

## Impact Assessment

| Scope                  | Impact                                                         |
|------------------------|----------------------------------------------------------------|
| Database volumes full  | All write operations fail; services return 500 errors          |
| Node disk pressure     | Kubelet evicts pods; new pods cannot be scheduled              |
| Log volumes full       | Application logs stop; debugging becomes impossible            |
| Kafka log dirs full    | Kafka brokers go offline; event processing halts               |
| Container ephemeral    | Pods are evicted; restarts fail if disk remains full           |

---

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export GRAFANA_URL=https://grafana.clario360.io
```

- `kubectl` configured with access to the `clario360` namespace
- `psql` client installed (PostgreSQL 15+)
- Appropriate K8s RBAC permissions (`clario360-operator` role or higher)
- Access to Grafana dashboards at `https://grafana.clario360.io`

---

## Diagnosis Steps

### Step 1: Identify nodes with disk pressure

```bash
kubectl get nodes -o custom-columns="NAME:.metadata.name,DISK_PRESSURE:.status.conditions[?(@.type=='DiskPressure')].status,READY:.status.conditions[?(@.type=='Ready')].status"
```

### Step 2: Inspect a specific node's conditions and capacity

```bash
kubectl describe node <NODE_NAME> | grep -A 5 "Conditions:"
```

```bash
kubectl describe node <NODE_NAME> | grep -A 10 "Allocated resources:"
```

### Step 3: Check disk usage on a node via a debug pod

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "df -h"
```

### Step 4: List PVCs and their usage in the clario360 namespace

```bash
kubectl get pvc -n clario360 -o custom-columns="NAME:.metadata.name,STATUS:.status.phase,CAPACITY:.status.capacity.storage,STORAGECLASS:.spec.storageClassName"
```

### Step 5: Check actual PVC usage with kubelet metrics

```bash
kubectl get --raw /api/v1/nodes/<NODE_NAME>/proxy/stats/summary | jq '.pods[] | select(.podRef.namespace=="clario360") | {pod: .podRef.name, volumes: [.volume[]? | {name: .name, usedBytes: .usedBytes, capacityBytes: .capacityBytes, pvcRef: .pvcRef?}]}'
```

### Step 6: Check PostgreSQL data directory usage

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- df -h /var/lib/postgresql/data
```

### Step 7: Check database sizes

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "SELECT datname, pg_size_pretty(pg_database_size(datname)) AS size FROM pg_database ORDER BY pg_database_size(datname) DESC;"
```

### Step 8: Check table-level disk usage in each database

```bash
for DB in platform_core cyber_db data_db acta_db lex_db visus_db; do
  echo "=== ${DB} ==="
  kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d ${DB} -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS total_size FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema') ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC LIMIT 10;"
done
```

### Step 9: Check Kafka log directory sizes

```bash
kubectl exec -it -n kafka $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- df -h /var/lib/kafka/data
```

```bash
kubectl exec -it -n kafka $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- du -sh /var/lib/kafka/data/*
```

### Step 10: Check container log sizes on the node

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "du -sh /var/log/containers/* | sort -rh | head -20"
```

### Step 11: Check for evicted pods

```bash
kubectl get pods -n clario360 --field-selector=status.phase=Failed -o custom-columns="NAME:.metadata.name,REASON:.status.reason,MESSAGE:.status.message" | grep -i evict
```

---

## Resolution Steps

### Option A: Clean old container logs (immediate relief)

**Step 1**: Identify pods with excessive logs:

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "du -sh /var/log/pods/clario360_*/* | sort -rh | head -20"
```

**Step 2**: Truncate oversized container log files on the affected node:

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "find /var/log/pods/clario360_* -name '*.log' -size +100M -exec truncate -s 0 {} \;"
```

**Step 3**: Verify log rotation is configured. Check the kubelet config:

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "cat /var/lib/kubelet/config.yaml | grep -A 2 containerLog"
```

### Option B: Expand a PVC (if the StorageClass supports volume expansion)

**Step 1**: Verify the StorageClass allows expansion:

```bash
kubectl get storageclass -o custom-columns="NAME:.metadata.name,PROVISIONER:.provisioner,ALLOW_EXPANSION:.allowVolumeExpansion"
```

**Step 2**: Patch the PVC to increase size (example: PostgreSQL PVC from 50Gi to 100Gi):

```bash
kubectl patch pvc postgresql-data -n clario360 --type merge -p '{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}'
```

**Step 3**: Monitor the resize operation:

```bash
kubectl get pvc postgresql-data -n clario360 -w
```

**Step 4**: Verify the new size is reflected in the pod:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- df -h /var/lib/postgresql/data
```

### Option C: Clean old database data

**Step 1**: Remove old audit log entries beyond the retention period (e.g., older than 90 days):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';"
```

**Step 2**: Remove expired sessions:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "DELETE FROM sessions WHERE expires_at < NOW();"
```

**Step 3**: Reclaim space with VACUUM FULL on the affected tables:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "VACUUM FULL VERBOSE audit_logs;"
```

**Step 4**: Verify reclaimed space:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "SELECT pg_size_pretty(pg_total_relation_size('audit_logs'));"
```

### Option D: Clean Kafka log segments

**Step 1**: Check current retention settings:

```bash
kubectl exec -it -n kafka $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- kafka-configs.sh --bootstrap-server localhost:9092 --entity-type topics --entity-name audit-events --describe | grep retention
```

**Step 2**: Temporarily reduce retention to purge old segments (e.g., 1 hour):

```bash
kubectl exec -it -n kafka $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- kafka-configs.sh --bootstrap-server localhost:9092 --entity-type topics --entity-name audit-events --alter --add-config retention.ms=3600000
```

**Step 3**: Wait for log cleaner to run (typically 5 minutes), then restore retention:

```bash
kubectl exec -it -n kafka $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- kafka-configs.sh --bootstrap-server localhost:9092 --entity-type topics --entity-name audit-events --alter --add-config retention.ms=604800000
```

### Option E: Add log rotation configuration

**Step 1**: Verify each service deployment has log rotation limits:

```bash
kubectl get deployment -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].resources.limits}{"\n"}{end}'
```

**Step 2**: Patch deployments to add ephemeral storage limits (example for api-gateway):

```bash
kubectl patch deployment api-gateway -n clario360 --type strategic -p '{"spec":{"template":{"spec":{"containers":[{"name":"api-gateway","resources":{"limits":{"ephemeral-storage":"2Gi"},"requests":{"ephemeral-storage":"1Gi"}}}]}}}}'
```

**Step 3**: Repeat for all services:

```bash
for SVC in iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl patch deployment ${SVC} -n clario360 --type strategic -p '{"spec":{"template":{"spec":{"containers":[{"name":"'${SVC}'","resources":{"limits":{"ephemeral-storage":"2Gi"},"requests":{"ephemeral-storage":"1Gi"}}}]}}}}'
done
```

### Option F: Delete evicted pods to reclaim resources

```bash
kubectl delete pods -n clario360 --field-selector=status.phase=Failed
```

---

## Verification

### Step 1: Confirm node disk pressure is cleared

```bash
kubectl get nodes -o custom-columns="NAME:.metadata.name,DISK_PRESSURE:.status.conditions[?(@.type=='DiskPressure')].status"
```

Expected: All nodes show `DiskPressure: False`.

### Step 2: Confirm PVC usage is within limits

```bash
kubectl get --raw /api/v1/nodes/<NODE_NAME>/proxy/stats/summary | jq '.pods[] | select(.podRef.namespace=="clario360") | .volume[]? | select(.pvcRef != null) | {pvc: .pvcRef.name, usedGB: (.usedBytes / 1073741824 * 100 | floor / 100), capacityGB: (.capacityBytes / 1073741824 * 100 | floor / 100)}'
```

Expected: No PVC exceeds 80% utilization.

### Step 3: Confirm all pods are running

```bash
kubectl get pods -n clario360 -o wide
```

Expected: All pods in `Running` state with no evictions.

### Step 4: Confirm services are healthy

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

### Step 5: Confirm PostgreSQL can accept writes

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "CREATE TABLE _disk_test (id int); DROP TABLE _disk_test; SELECT 'WRITE_OK';"
```

### Step 6: Check Grafana disk dashboard

Open `${GRAFANA_URL}/d/cluster-overview` and verify disk utilization is below 80% on all nodes.

---

## Post-Incident Checklist

- [ ] Document the root cause (which volume/node ran out of space and why)
- [ ] Verify disk space alerts are firing correctly in Prometheus/Alertmanager
- [ ] Confirm alert thresholds are appropriate (recommend: warn at 75%, critical at 85%)
- [ ] Review log rotation policies across all services
- [ ] Review data retention policies (audit logs, sessions, Kafka topics)
- [ ] Verify PVC auto-expansion is enabled where supported
- [ ] Update capacity planning spreadsheet with current disk growth rates
- [ ] Schedule VACUUM FULL for any tables that had large deletions
- [ ] Create or update Jira ticket for long-term capacity needs
- [ ] Update this runbook with any new patterns discovered

---

## Related Links

- [OP-007: Database Maintenance](../operations/OP-007-database-maintenance.md) -- VACUUM and REINDEX procedures
- [OP-009: Log Management](../operations/OP-009-log-management.md) -- Log rotation and retention policies
- [SC-004: Node Pool Scaling](../scaling/SC-004-node-pool-scaling.md) -- Adding K8s nodes
- [SC-005: Capacity Planning](../scaling/SC-005-capacity-planning.md) -- Capacity planning guide
- [TS-007: High CPU Usage](../troubleshooting/TS-007-high-cpu-usage.md) -- Related resource issues
- Grafana Cluster Overview: `${GRAFANA_URL}/d/cluster-overview`
- Kubernetes PVC documentation: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
