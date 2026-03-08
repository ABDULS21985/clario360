# SC-001: Horizontal Pod Scaling

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | SC-001                                     |
| **Title**        | Horizontal Pod Scaling                     |
| **Category**     | Scaling                                    |
| **Severity**     | Medium                                     |
| **Author**       | Platform Engineering                       |
| **Created**      | 2026-03-08                                 |
| **Last Updated** | 2026-03-08                                 |
| **Review Cycle** | Quarterly                                  |
| **Platform**     | GCP (GKE)                                  |
| **Namespace**    | clario360                                  |

---

## Summary

This runbook covers horizontal scaling of Clario 360 platform services running on GKE. It includes manual scaling via `kubectl scale`, configuring and tuning Horizontal Pod Autoscalers (HPA), per-service scaling recommendations, and post-scaling verification procedures. Use this runbook when services are experiencing increased latency, high CPU/memory utilization, or ahead of anticipated traffic increases.

---

## Prerequisites

- `kubectl` CLI configured with cluster credentials
- `gcloud` CLI authenticated with appropriate IAM roles (`roles/container.developer` or higher)
- Access to the `clario360` namespace
- Metrics Server deployed in the cluster (required for HPA)
- Prometheus/Grafana dashboards available for monitoring
- Familiarity with current service deployment topology

### Verify Cluster Access

```bash
# Authenticate with GKE cluster
gcloud container clusters get-credentials clario360-cluster \
  --region us-central1 \
  --project clario360-prod

# Verify namespace access
kubectl get deployments -n clario360
```

---

## Per-Service Scaling Recommendations

| Service                | Min Replicas | Max Replicas | CPU Target | Memory Target | Notes                                      |
|------------------------|-------------|-------------|------------|---------------|--------------------------------------------|
| api-gateway            | 3           | 20          | 60%        | 70%           | Front-facing; scale aggressively            |
| iam-service            | 2           | 10          | 65%        | 70%           | Auth-critical; keep headroom                |
| audit-service          | 2           | 8           | 70%        | 75%           | Write-heavy; monitor disk I/O               |
| workflow-engine        | 2           | 12          | 60%        | 70%           | Stateful workflows; scale with caution      |
| notification-service   | 2           | 10          | 65%        | 70%           | WebSocket connections; check connection count|
| cyber-service          | 2           | 10          | 60%        | 70%           | Scan workloads are bursty                   |
| data-service           | 2           | 10          | 65%        | 75%           | ETL pipelines; memory-intensive             |
| acta-service           | 2           | 8           | 70%        | 75%           | Document processing; CPU-bound              |
| lex-service            | 2           | 8           | 65%        | 70%           | Regulatory lookups; mostly I/O-bound        |
| visus-service          | 2           | 8           | 65%        | 70%           | Reporting; bursty during business hours     |

---

## Procedure

### Step 1: Assess Current State

Check the current replica count and resource utilization for all services.

```bash
# List all deployments with current replica counts
kubectl get deployments -n clario360 -o wide

# Check current pod resource utilization
kubectl top pods -n clario360 --sort-by=cpu

# Check current HPA status for all services
kubectl get hpa -n clario360

# Inspect a specific service's deployment details
kubectl describe deployment api-gateway -n clario360
```

### Step 2: Manual Scaling via kubectl

Use manual scaling when you need immediate capacity increase and cannot wait for HPA to react.

```bash
# Scale a specific service (example: api-gateway to 6 replicas)
kubectl scale deployment api-gateway -n clario360 --replicas=6

# Scale multiple services at once
kubectl scale deployment api-gateway -n clario360 --replicas=6
kubectl scale deployment iam-service -n clario360 --replicas=4
kubectl scale deployment workflow-engine -n clario360 --replicas=4

# Verify the scaling event
kubectl rollout status deployment/api-gateway -n clario360

# Confirm new pods are running
kubectl get pods -n clario360 -l app=api-gateway -o wide
```

### Step 3: Configure Horizontal Pod Autoscaler (HPA)

#### 3a: Create HPA with CPU-Based Scaling

```bash
# Create HPA for api-gateway (CPU-based)
kubectl autoscale deployment api-gateway -n clario360 \
  --min=3 \
  --max=20 \
  --cpu-percent=60
```

#### 3b: Apply HPA with CPU and Memory Targets (YAML)

For more granular control, apply an HPA manifest with both CPU and memory metrics.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
      - type: Pods
        value: 4
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF
```

#### 3c: Apply HPA for All Platform Services

Repeat for each service, adjusting thresholds per the recommendations table above.

```bash
# iam-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: iam-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: iam-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 65
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 3
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# audit-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: audit-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: audit-service
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 75
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# workflow-engine HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: workflow-engine-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: workflow-engine
  minReplicas: 2
  maxReplicas: 12
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 3
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# notification-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: notification-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: notification-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 65
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 3
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# cyber-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: cyber-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cyber-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 3
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# data-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: data-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: data-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 65
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 75
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 3
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# acta-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: acta-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: acta-service
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 75
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# lex-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: lex-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: lex-service
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 65
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF

# visus-service HPA
cat <<'EOF' | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: visus-service-hpa
  namespace: clario360
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: visus-service
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 65
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
      selectPolicy: Min
EOF
```

### Step 4: Tune HPA Behavior

#### 4a: Adjust Scaling Velocity

If the HPA is scaling too slowly during traffic spikes, reduce the stabilization window and increase the scale-up policy.

```bash
kubectl patch hpa api-gateway-hpa -n clario360 --type merge -p '{
  "spec": {
    "behavior": {
      "scaleUp": {
        "stabilizationWindowSeconds": 30,
        "policies": [
          {"type": "Percent", "value": 100, "periodSeconds": 30},
          {"type": "Pods", "value": 6, "periodSeconds": 30}
        ],
        "selectPolicy": "Max"
      }
    }
  }
}'
```

#### 4b: Prevent Flapping (Aggressive Scale-Down Prevention)

If pods are frequently scaling up and down, increase the scale-down stabilization window.

```bash
kubectl patch hpa api-gateway-hpa -n clario360 --type merge -p '{
  "spec": {
    "behavior": {
      "scaleDown": {
        "stabilizationWindowSeconds": 600,
        "policies": [
          {"type": "Percent", "value": 10, "periodSeconds": 300}
        ],
        "selectPolicy": "Min"
      }
    }
  }
}'
```

### Step 5: Verify Scaling Events and Pod Distribution

```bash
# Check HPA status and current metrics
kubectl get hpa -n clario360

# Detailed HPA info including events
kubectl describe hpa api-gateway-hpa -n clario360

# Verify pods are distributed across nodes
kubectl get pods -n clario360 -l app=api-gateway -o wide

# Check pod distribution across nodes (all services)
kubectl get pods -n clario360 -o wide --sort-by='.spec.nodeName'

# Verify no pods are in Pending state
kubectl get pods -n clario360 --field-selector=status.phase=Pending

# Check recent scaling events
kubectl get events -n clario360 --sort-by='.lastTimestamp' --field-selector reason=SuccessfulRescale

# Verify pod anti-affinity is spreading pods across nodes
kubectl get pods -n clario360 -l app=api-gateway -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.nodeName}{"\n"}{end}'
```

### Step 6: Validate Pod Disruption Budgets

Ensure PDBs are in place before scaling to maintain availability during disruptions.

```bash
# Check existing PDBs
kubectl get pdb -n clario360

# Create PDB for api-gateway if not present
cat <<'EOF' | kubectl apply -f -
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: api-gateway-pdb
  namespace: clario360
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: api-gateway
EOF

# Create PDBs for all critical services
for service in iam-service audit-service workflow-engine notification-service; do
cat <<EOF | kubectl apply -f -
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ${service}-pdb
  namespace: clario360
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: ${service}
EOF
done
```

### Step 7: Load Test Validation After Scaling

Run a load test to validate that the scaled deployment handles the expected traffic.

```bash
# Port-forward to the api-gateway for local testing (if no external LB)
kubectl port-forward svc/api-gateway -n clario360 8080:80 &

# Run a basic load test with hey (install: go install github.com/rakyll/hey@latest)
hey -n 10000 -c 200 -z 60s \
  -H "Authorization: Bearer ${TEST_TOKEN}" \
  http://localhost:8080/api/v1/health

# Monitor HPA scaling in real-time during the load test
kubectl get hpa -n clario360 -w

# Monitor pod count in real-time during the load test
kubectl get pods -n clario360 -l app=api-gateway -w

# Check resource utilization during load test
kubectl top pods -n clario360 -l app=api-gateway

# After the load test, verify all pods are healthy
kubectl get pods -n clario360 -l app=api-gateway -o wide
```

### Step 8: Rollback (If Scaling Causes Issues)

```bash
# Scale back to the previous replica count
kubectl scale deployment api-gateway -n clario360 --replicas=3

# Delete HPA if autoscaling is causing problems
kubectl delete hpa api-gateway-hpa -n clario360

# Check for OOMKilled pods that may indicate resource limits are too low
kubectl get pods -n clario360 -o json | \
  kubectl get pods -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .status.containerStatuses[*]}{.lastState.terminated.reason}{end}{"\n"}{end}'
```

---

## Verification

After completing the scaling procedure, confirm the following:

1. All pods are in `Running` state with `Ready` condition:
   ```bash
   kubectl get pods -n clario360 -l app=api-gateway
   ```

2. HPA shows `TARGETS` with actual metrics (not `<unknown>`):
   ```bash
   kubectl get hpa -n clario360
   ```

3. Service endpoints include all new pods:
   ```bash
   kubectl get endpoints api-gateway -n clario360
   ```

4. No error logs from newly scaled pods:
   ```bash
   kubectl logs -n clario360 -l app=api-gateway --tail=50 --since=5m
   ```

5. Prometheus metrics confirm requests are distributed across pods:
   ```bash
   # Via Grafana or PromQL:
   # sum(rate(http_requests_total{namespace="clario360",service="api-gateway"}[5m])) by (pod)
   ```

---

## Related Links

| Document                  | Link                                             |
|---------------------------|--------------------------------------------------|
| Node Pool Scaling         | [SC-004-node-pool-scaling.md](SC-004-node-pool-scaling.md) |
| Capacity Planning         | [SC-005-capacity-planning.md](SC-005-capacity-planning.md) |
| Database Scaling          | [SC-002-database-scaling.md](SC-002-database-scaling.md)   |
| Kafka Scaling             | [SC-003-kafka-scaling.md](SC-003-kafka-scaling.md)         |
| GKE Autoscaling Docs      | https://cloud.google.com/kubernetes-engine/docs/concepts/horizontalpodautoscaler |
| Kubernetes HPA v2 API     | https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/ |
