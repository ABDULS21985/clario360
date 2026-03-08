# SC-004: GKE Node Pool Scaling

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | SC-004                                     |
| **Title**        | GKE Node Pool Scaling                      |
| **Category**     | Scaling                                    |
| **Severity**     | High                                       |
| **Author**       | Platform Engineering                       |
| **Created**      | 2026-03-08                                 |
| **Last Updated** | 2026-03-08                                 |
| **Review Cycle** | Quarterly                                  |
| **Platform**     | GCP (GKE)                                  |
| **Cluster**      | clario360-cluster                          |
| **Region**       | us-central1                                |

---

## Summary

This runbook covers scaling GKE node pools for the Clario 360 platform. It includes scaling existing node pools, adding new node pools with different machine types, configuring the cluster autoscaler, draining and cordoning nodes for maintenance, and verifying pod scheduling after node additions. Node pool scaling is the foundation that enables horizontal pod scaling (SC-001) -- pods cannot be scheduled if there are insufficient nodes.

### Current Node Pool Configuration

| Node Pool        | Machine Type    | Min Nodes | Max Nodes | Purpose                                |
|------------------|----------------|-----------|-----------|----------------------------------------|
| default-pool     | e2-standard-4  | 3         | 10        | General platform services              |
| high-memory-pool | e2-highmem-8   | 1         | 5         | PostgreSQL, Kafka, data-service        |
| compute-pool     | c2-standard-8  | 0         | 5         | CPU-intensive workloads (cyber-service)|

---

## Prerequisites

- `gcloud` CLI authenticated with `roles/container.admin` or `roles/container.clusterAdmin`
- `kubectl` CLI configured with cluster credentials
- Sufficient GCP quota for the desired machine types in `us-central1`
- Understanding of current workload resource requirements
- Change management approval for production changes

### Verify Access

```bash
# Authenticate and configure
gcloud auth login
gcloud config set project clario360-prod
gcloud config set compute/region us-central1

# Get cluster credentials
gcloud container clusters get-credentials clario360-cluster \
  --region us-central1 \
  --project clario360-prod

# Verify current node pools
gcloud container node-pools list \
  --cluster clario360-cluster \
  --region us-central1 \
  --format="table(name, config.machineType, autoscaling.minNodeCount, autoscaling.maxNodeCount, status)"

# Verify current node status
kubectl get nodes -o wide
```

---

## Procedure

### Step 1: Assess Current Node Utilization

```bash
# Check node resource utilization
kubectl top nodes

# Detailed node capacity and allocations
kubectl describe nodes | grep -A 6 "Allocated resources"

# Check for pods in Pending state (indicates node pressure)
kubectl get pods --all-namespaces --field-selector=status.phase=Pending

# Check node conditions (MemoryPressure, DiskPressure, PIDPressure)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .status.conditions[*]}{.type}={.status}{" "}{end}{"\n"}{end}'

# Check GCP quota availability for the region
gcloud compute regions describe us-central1 \
  --format="table(quotas.metric, quotas.limit, quotas.usage)" \
  --flatten="quotas"

# List current node count per pool
gcloud container node-pools list \
  --cluster clario360-cluster \
  --region us-central1 \
  --format="table(name, config.machineType, initialNodeCount, autoscaling.enabled, autoscaling.minNodeCount, autoscaling.maxNodeCount)"
```

### Step 2: Scale Existing Node Pool

#### 2a: Resize a Node Pool (Manual)

```bash
# Scale the default-pool from current size to 6 nodes
gcloud container clusters resize clario360-cluster \
  --region us-central1 \
  --node-pool default-pool \
  --num-nodes 6 \
  --quiet

# Wait for the resize operation to complete
gcloud container operations list \
  --region us-central1 \
  --filter="TYPE=RESIZE_NODE_POOL AND STATUS=RUNNING" \
  --format="table(name, status, startTime)"

# Verify new nodes have joined the cluster
kubectl get nodes -l cloud.google.com/gke-nodepool=default-pool -o wide

# Wait for all new nodes to be Ready
kubectl wait --for=condition=Ready node -l cloud.google.com/gke-nodepool=default-pool --timeout=300s
```

#### 2b: Update Autoscaling Limits for Existing Pool

```bash
# Update autoscaling limits for the default pool
gcloud container clusters update clario360-cluster \
  --region us-central1 \
  --node-pool default-pool \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 15

# Update autoscaling limits for the high-memory pool
gcloud container clusters update clario360-cluster \
  --region us-central1 \
  --node-pool high-memory-pool \
  --enable-autoscaling \
  --min-nodes 2 \
  --max-nodes 8

# Update autoscaling limits for the compute pool
gcloud container clusters update clario360-cluster \
  --region us-central1 \
  --node-pool compute-pool \
  --enable-autoscaling \
  --min-nodes 0 \
  --max-nodes 8

# Verify the updated autoscaling configuration
gcloud container node-pools describe default-pool \
  --cluster clario360-cluster \
  --region us-central1 \
  --format="yaml(autoscaling)"
```

### Step 3: Add New Node Pool with Different Machine Types

#### 3a: Add a High-Memory Node Pool

```bash
# Create a new high-memory node pool for database workloads
gcloud container node-pools create high-memory-pool-v2 \
  --cluster clario360-cluster \
  --region us-central1 \
  --machine-type e2-highmem-16 \
  --num-nodes 2 \
  --min-nodes 1 \
  --max-nodes 4 \
  --enable-autoscaling \
  --disk-type pd-ssd \
  --disk-size 200 \
  --node-labels="workload-type=database,tier=data" \
  --node-taints="dedicated=database:NoSchedule" \
  --metadata disable-legacy-endpoints=true \
  --scopes="gke-default,storage-full" \
  --max-pods-per-node 32 \
  --enable-autorepair \
  --enable-autoupgrade

# Verify the new node pool
gcloud container node-pools describe high-memory-pool-v2 \
  --cluster clario360-cluster \
  --region us-central1

# Verify nodes have the correct labels and taints
kubectl get nodes -l workload-type=database -o wide
kubectl describe nodes -l workload-type=database | grep -A 3 "Taints"
```

#### 3b: Add a CPU-Optimized Node Pool

```bash
# Create a CPU-optimized pool for compute-intensive services
gcloud container node-pools create compute-pool-v2 \
  --cluster clario360-cluster \
  --region us-central1 \
  --machine-type c2-standard-16 \
  --num-nodes 1 \
  --min-nodes 0 \
  --max-nodes 6 \
  --enable-autoscaling \
  --disk-type pd-ssd \
  --disk-size 100 \
  --node-labels="workload-type=compute,tier=processing" \
  --node-taints="dedicated=compute:NoSchedule" \
  --metadata disable-legacy-endpoints=true \
  --scopes="gke-default" \
  --max-pods-per-node 64 \
  --enable-autorepair \
  --enable-autoupgrade

# Verify the new node pool
kubectl get nodes -l workload-type=compute -o wide
```

#### 3c: Add a Spot/Preemptible Node Pool for Non-Critical Workloads

```bash
# Create a spot node pool for cost-effective batch processing
gcloud container node-pools create spot-pool \
  --cluster clario360-cluster \
  --region us-central1 \
  --machine-type e2-standard-8 \
  --num-nodes 0 \
  --min-nodes 0 \
  --max-nodes 10 \
  --enable-autoscaling \
  --spot \
  --disk-type pd-standard \
  --disk-size 100 \
  --node-labels="workload-type=batch,tier=spot" \
  --node-taints="cloud.google.com/gke-spot=true:NoSchedule" \
  --metadata disable-legacy-endpoints=true \
  --scopes="gke-default" \
  --max-pods-per-node 64 \
  --enable-autorepair \
  --enable-autoupgrade
```

### Step 4: Configure Cluster Autoscaler

#### 4a: Enable and Configure Cluster Autoscaler

```bash
# Enable cluster autoscaler with profile
gcloud container clusters update clario360-cluster \
  --region us-central1 \
  --enable-autoscaling \
  --autoscaling-profile optimize-utilization

# For balanced (default) profile (scale down less aggressively):
# gcloud container clusters update clario360-cluster \
#   --region us-central1 \
#   --enable-autoscaling \
#   --autoscaling-profile balanced
```

#### 4b: Configure Autoscaler Parameters via Resource Limits

```bash
# Set cluster-wide resource limits for the autoscaler
gcloud container clusters update clario360-cluster \
  --region us-central1 \
  --max-cpu 200 \
  --max-memory 800 \
  --min-cpu 12 \
  --min-memory 48
```

#### 4c: Verify Autoscaler Status

```bash
# Check cluster autoscaler status
kubectl get configmap cluster-autoscaler-status -n kube-system -o yaml

# Check autoscaler events
kubectl get events -n kube-system --field-selector source=cluster-autoscaler --sort-by='.lastTimestamp'

# Check for scale-up/scale-down decisions
kubectl logs -n kube-system -l app=cluster-autoscaler --tail=100 | grep -E "Scale-(up|down)"
```

#### 4d: Configure Pod Priority for Autoscaler Decisions

```bash
# Create PriorityClasses for workload tiering
cat <<'EOF' | kubectl apply -f -
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: platform-critical
value: 1000000
globalDefault: false
description: "Critical platform services (api-gateway, iam-service)"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: platform-standard
value: 500000
globalDefault: true
description: "Standard platform services"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: platform-batch
value: 100000
globalDefault: false
description: "Batch and non-critical workloads"
EOF

# Assign priority classes to deployments
kubectl patch deployment api-gateway -n clario360 --type merge -p '{
  "spec": {"template": {"spec": {"priorityClassName": "platform-critical"}}}
}'

kubectl patch deployment iam-service -n clario360 --type merge -p '{
  "spec": {"template": {"spec": {"priorityClassName": "platform-critical"}}}
}'
```

### Step 5: Drain and Cordon Nodes for Maintenance

#### 5a: Cordon a Node (Prevent New Scheduling)

```bash
# List nodes to identify the target
kubectl get nodes -o wide

# Cordon the node (prevents new pod scheduling)
kubectl cordon gke-clario360-cluster-default-pool-abc123-0

# Verify node is cordoned (SchedulingDisabled)
kubectl get nodes | grep SchedulingDisabled
```

#### 5b: Drain a Node (Evict All Pods)

```bash
# Drain the node gracefully with PDB respect
kubectl drain gke-clario360-cluster-default-pool-abc123-0 \
  --grace-period=60 \
  --timeout=300s \
  --ignore-daemonsets \
  --delete-emptydir-data \
  --force

# Monitor pod rescheduling
kubectl get pods -n clario360 -o wide --watch

# Verify all pods from the drained node have been rescheduled
kubectl get pods --all-namespaces --field-selector spec.nodeName=gke-clario360-cluster-default-pool-abc123-0
```

#### 5c: Uncordon a Node After Maintenance

```bash
# After maintenance, uncordon to resume scheduling
kubectl uncordon gke-clario360-cluster-default-pool-abc123-0

# Verify node is schedulable
kubectl get nodes | grep -v SchedulingDisabled
```

#### 5d: Rolling Drain of Entire Node Pool

```bash
# Drain all nodes in a pool one at a time for zero-downtime maintenance
NODES=$(kubectl get nodes -l cloud.google.com/gke-nodepool=default-pool -o jsonpath='{.items[*].metadata.name}')

for node in ${NODES}; do
  echo "=== Draining ${node} ==="
  kubectl cordon ${node}
  kubectl drain ${node} \
    --grace-period=60 \
    --timeout=300s \
    --ignore-daemonsets \
    --delete-emptydir-data \
    --force

  echo "Waiting for pods to stabilize..."
  sleep 30

  # Verify all clario360 pods are running
  PENDING=$(kubectl get pods -n clario360 --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
  if [ "${PENDING}" -gt 0 ]; then
    echo "WARNING: ${PENDING} pods still pending. Waiting additional 60s..."
    sleep 60
  fi

  kubectl uncordon ${node}
  echo "=== Completed ${node} ==="
  sleep 10
done
```

### Step 6: Verify Pod Scheduling After Node Addition

```bash
# Check that all pods are scheduled and running
kubectl get pods -n clario360 -o wide

# Verify pod distribution across nodes
kubectl get pods -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.nodeName}{"\n"}{end}' | sort -k2

# Check for any scheduling failures
kubectl get events -n clario360 --field-selector reason=FailedScheduling --sort-by='.lastTimestamp'

# Verify node affinity and tolerations are working
kubectl get pods -n clario360 -l workload-type=database -o wide

# Check that services are distributing across zones for HA
kubectl get pods -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.nodeName}{"\t"}{.metadata.labels.app}{"\n"}{end}' | sort -k3

# Verify topology spread constraints
kubectl get nodes --show-labels | grep topology.kubernetes.io/zone
```

### Step 7: Remove Old Node Pool (After Migration)

```bash
# Verify no pods are running on the old pool's nodes
OLD_POOL_NODES=$(kubectl get nodes -l cloud.google.com/gke-nodepool=high-memory-pool -o jsonpath='{.items[*].metadata.name}')
for node in ${OLD_POOL_NODES}; do
  echo "=== Pods on ${node} ==="
  kubectl get pods --all-namespaces --field-selector spec.nodeName=${node} --no-headers
done

# Drain all nodes in the old pool
for node in ${OLD_POOL_NODES}; do
  kubectl drain ${node} \
    --grace-period=60 \
    --timeout=300s \
    --ignore-daemonsets \
    --delete-emptydir-data \
    --force
done

# Delete the old node pool
gcloud container node-pools delete high-memory-pool \
  --cluster clario360-cluster \
  --region us-central1 \
  --quiet

# Verify the pool is removed
gcloud container node-pools list \
  --cluster clario360-cluster \
  --region us-central1
```

---

## Verification

After completing node pool scaling, confirm the following:

1. All nodes are in `Ready` state:
   ```bash
   kubectl get nodes -o wide
   ```

2. No pods are in `Pending` state:
   ```bash
   kubectl get pods --all-namespaces --field-selector=status.phase=Pending
   ```

3. Node pools have correct autoscaling configuration:
   ```bash
   gcloud container node-pools list \
     --cluster clario360-cluster \
     --region us-central1 \
     --format="table(name, config.machineType, autoscaling.enabled, autoscaling.minNodeCount, autoscaling.maxNodeCount)"
   ```

4. All Clario 360 services are running:
   ```bash
   kubectl get deployments -n clario360
   ```

5. Cluster autoscaler is functioning:
   ```bash
   kubectl get configmap cluster-autoscaler-status -n kube-system -o yaml
   ```

6. Resource utilization is balanced across nodes:
   ```bash
   kubectl top nodes
   ```

---

## Related Links

| Document                  | Link                                             |
|---------------------------|--------------------------------------------------|
| Horizontal Pod Scaling    | [SC-001-horizontal-scaling.md](SC-001-horizontal-scaling.md) |
| Database Scaling          | [SC-002-database-scaling.md](SC-002-database-scaling.md)     |
| Kafka Scaling             | [SC-003-kafka-scaling.md](SC-003-kafka-scaling.md)           |
| Capacity Planning         | [SC-005-capacity-planning.md](SC-005-capacity-planning.md)   |
| GKE Cluster Autoscaler    | https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-autoscaler |
| GKE Node Pool Management  | https://cloud.google.com/kubernetes-engine/docs/how-to/node-pools |
