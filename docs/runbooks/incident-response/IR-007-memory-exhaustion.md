# IR-007: OOMKilled Pods — Memory Exhaustion

| Field            | Value                                    |
|------------------|------------------------------------------|
| Runbook ID       | IR-007                                   |
| Title            | OOMKilled Pods — Memory Exhaustion       |
| Severity         | P2 — High                               |
| Owner            | Platform Team                            |
| Last Updated     | 2026-03-08                               |
| Review Frequency | Quarterly                                |
| Approver         | Platform Lead                            |

---

## Summary

This runbook covers diagnosis and resolution of memory exhaustion issues in the Clario 360 platform. Memory exhaustion typically manifests as pods being OOMKilled by the Linux kernel, node-level memory pressure causing pod evictions, or gradual memory leaks degrading service performance. OOMKilled pods restart automatically but repeated occurrences indicate insufficient resource limits or application-level memory leaks.

---

## Symptoms

- **Alerts**: `KubePodCrashLooping`, `KubeContainerOOMKilled`, `NodeMemoryPressure`, `KubeMemoryOvercommit`
- Pod status shows `OOMKilled` in `kubectl describe pod` last state
- Pods in `CrashLoopBackOff` with `OOMKilled` as the last termination reason
- `kubectl describe node` shows `MemoryPressure` condition as `True`
- Grafana dashboard `Cluster Overview` (`/d/cluster-overview`) shows memory utilization above 85%
- Grafana dashboard `Service Health` (`/d/service-health`) shows steadily increasing memory usage over time (leak pattern)
- Application response times degrade as the Go runtime spends more time in GC
- Services become unresponsive before being killed

---

## Impact Assessment

| Scope                    | Impact                                                          |
|--------------------------|-----------------------------------------------------------------|
| Single pod OOMKilled     | Brief service interruption; auto-restart within seconds         |
| Repeated OOMKill (CrashLoop) | Service degraded; pod restart backoff up to 5 minutes      |
| Node memory pressure     | Kubelet evicts pods; workloads rescheduled to other nodes       |
| All replicas OOMKilled   | Full service outage; dependent services return errors           |
| Memory leak (gradual)    | Slow degradation; increasing GC pause times; eventual OOMKill  |
| PostgreSQL OOMKilled     | Database connections dropped; potential data corruption risk    |

---

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export GRAFANA_URL=https://grafana.clario360.io
```

- `kubectl` configured with access to the `clario360` namespace
- `kubectl top` requires metrics-server to be installed
- Access to Grafana dashboards at `https://grafana.clario360.io`
- Appropriate K8s RBAC permissions (`clario360-operator` role or higher)

---

## Diagnosis Steps

### Step 1: Identify OOMKilled pods

```bash
kubectl get pods -n clario360 -o json | jq -r '.items[] | select(.status.containerStatuses[]?.lastState.terminated.reason == "OOMKilled") | "\(.metadata.name)\t\(.status.containerStatuses[0].lastState.terminated.reason)\t\(.status.containerStatuses[0].lastState.terminated.finishedAt)\t\(.status.containerStatuses[0].restartCount) restarts"'
```

### Step 2: Get detailed OOM information for a specific pod

```bash
kubectl describe pod <POD_NAME> -n clario360 | grep -A 20 "Last State:"
```

### Step 3: Check current memory usage across all pods

```bash
kubectl top pods -n clario360 --sort-by=memory
```

### Step 4: Compare actual memory usage against limits

```bash
kubectl get pods -n clario360 -o json | jq -r '.items[] | "\(.metadata.name)\tLimit: \(.spec.containers[0].resources.limits.memory // "none")\tRequest: \(.spec.containers[0].resources.requests.memory // "none")"'
```

### Step 5: Check node-level memory pressure

```bash
kubectl get nodes -o custom-columns="NAME:.metadata.name,MEMORY_PRESSURE:.status.conditions[?(@.type=='MemoryPressure')].status,READY:.status.conditions[?(@.type=='Ready')].status"
```

### Step 6: Check node memory capacity and allocatable

```bash
kubectl describe nodes | grep -A 5 "Allocated resources:" | grep -A 2 "memory"
```

### Step 7: Check node-level memory usage

```bash
kubectl top nodes
```

### Step 8: Check memory usage trend for a specific service in Grafana

Open the following URL to see the memory trend:

```
${GRAFANA_URL}/d/service-health?var-service=<SERVICE_NAME>&var-namespace=clario360
```

### Step 9: Check for memory leak pattern (Go services)

Look for steadily increasing RSS without corresponding workload increase:

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/heap?debug=1 2>/dev/null | head -30
```

### Step 10: Get Go runtime memory statistics

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/heap?debug=2 2>/dev/null | grep -E "^# runtime.MemStats"
```

### Step 11: Capture a heap profile for offline analysis

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/heap > /tmp/${SERVICE_NAME}-heap-$(date +%Y%m%d-%H%M%S).pb.gz
```

Analyze the heap profile locally:

```bash
go tool pprof -http=:8081 /tmp/${SERVICE_NAME}-heap-*.pb.gz
```

### Step 12: Check for evicted pods

```bash
kubectl get pods -n clario360 --field-selector=status.phase=Failed -o custom-columns="NAME:.metadata.name,REASON:.status.reason,MESSAGE:.status.message"
```

### Step 13: Check kernel OOM killer logs on the node

```bash
kubectl debug node/<NODE_NAME> -it --image=busybox -- sh -c "dmesg | grep -i 'oom\|out of memory\|killed process' | tail -20"
```

### Step 14: Check PostgreSQL memory usage if database pod is affected

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "SHOW shared_buffers; SHOW work_mem; SHOW effective_cache_size; SHOW maintenance_work_mem;"
```

---

## Resolution Steps

### Option A: Increase memory limits for a specific service (immediate)

**Step 1**: Check the current limits:

```bash
kubectl get deployment <SERVICE_NAME> -n clario360 -o jsonpath='{.spec.template.spec.containers[0].resources}' | jq .
```

**Step 2**: Increase memory limits (example: doubling from 512Mi to 1Gi):

```bash
kubectl patch deployment <SERVICE_NAME> -n clario360 --type strategic -p '{"spec":{"template":{"spec":{"containers":[{"name":"<SERVICE_NAME>","resources":{"limits":{"memory":"1Gi"},"requests":{"memory":"512Mi"}}}]}}}}'
```

**Step 3**: Wait for the rollout to complete:

```bash
kubectl rollout status deployment/<SERVICE_NAME> -n clario360 --timeout=120s
```

**Step 4**: Verify the new pod is running with updated limits:

```bash
kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].spec.containers[0].resources}' | jq .
```

### Option B: Restart the pod to clear a memory leak (temporary)

**Step 1**: Perform a rolling restart of the affected deployment:

```bash
kubectl rollout restart deployment/<SERVICE_NAME> -n clario360
```

**Step 2**: Monitor the rollout:

```bash
kubectl rollout status deployment/<SERVICE_NAME> -n clario360 --timeout=120s
```

**Step 3**: Verify memory usage dropped after restart:

```bash
kubectl top pod -n clario360 -l app=<SERVICE_NAME>
```

### Option C: Fix Go memory leak (requires code change)

**Step 1**: Capture heap profile before and after traffic:

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/heap > /tmp/heap-before.pb.gz
```

Wait for a period of traffic, then capture again:

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/heap > /tmp/heap-after.pb.gz
```

**Step 2**: Compare the two profiles to identify growing allocations:

```bash
go tool pprof -base /tmp/heap-before.pb.gz -http=:8081 /tmp/heap-after.pb.gz
```

**Step 3**: Check for goroutine leaks (common cause of Go memory leaks):

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE_NAME> -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/debug/pprof/goroutine?debug=2 > /tmp/goroutines.txt
```

```bash
head -5 /tmp/goroutines.txt
```

If goroutine count is very high (>1000), look for leaked goroutines in the output.

**Step 4**: Set GOMEMLIMIT environment variable to trigger GC earlier (temporary mitigation):

```bash
kubectl set env deployment/<SERVICE_NAME> -n clario360 GOMEMLIMIT=768MiB
```

### Option D: Address node-level memory pressure

**Step 1**: Identify pods consuming the most memory on the pressured node:

```bash
kubectl top pods -n clario360 --sort-by=memory | head -15
```

**Step 2**: Check if total memory requests exceed node capacity:

```bash
kubectl describe node <NODE_NAME> | grep -A 8 "Allocated resources:"
```

**Step 3**: If overcommitted, reduce resource requests or scale out to more nodes. Scale a specific deployment down temporarily:

```bash
kubectl scale deployment <LEAST_CRITICAL_SERVICE> -n clario360 --replicas=1
```

**Step 4**: To relieve pressure permanently, see [SC-004: Node Pool Scaling](../scaling/SC-004-node-pool-scaling.md).

### Option E: PostgreSQL memory tuning

**Step 1**: Check current PostgreSQL memory configuration:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "SELECT name, setting, unit, source FROM pg_settings WHERE name IN ('shared_buffers','work_mem','maintenance_work_mem','effective_cache_size','max_connections');"
```

**Step 2**: If work_mem is too high, reduce it (high values multiplied by max_connections can exhaust memory):

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "ALTER SYSTEM SET work_mem = '8MB'; SELECT pg_reload_conf();"
```

**Step 3**: Kill any long-running queries that may be holding memory:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'active' AND query_start < NOW() - INTERVAL '30 minutes' AND pid <> pg_backend_pid();"
```

### Option F: Delete evicted pods to clean up

```bash
kubectl delete pods -n clario360 --field-selector=status.phase=Failed
```

---

## Verification

### Step 1: Confirm no pods are OOMKilled or in CrashLoopBackOff

```bash
kubectl get pods -n clario360 -o wide | grep -E "OOMKilled|CrashLoopBackOff|Error"
```

Expected: No output (no unhealthy pods).

### Step 2: Confirm node memory pressure is cleared

```bash
kubectl get nodes -o custom-columns="NAME:.metadata.name,MEMORY_PRESSURE:.status.conditions[?(@.type=='MemoryPressure')].status"
```

Expected: All nodes show `MemoryPressure: False`.

### Step 3: Confirm current memory usage is healthy

```bash
kubectl top pods -n clario360 --sort-by=memory
```

Expected: No pod using more than 80% of its memory limit.

### Step 4: Confirm all services are healthy

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

### Step 5: Verify no restart count spike

```bash
kubectl get pods -n clario360 -o custom-columns="NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount,STATUS:.status.phase" --sort-by='.status.containerStatuses[0].restartCount'
```

Expected: Restart counts are stable (not increasing).

### Step 6: Check Grafana memory dashboard

Open `${GRAFANA_URL}/d/service-health` and verify memory usage trend is flat (not increasing) for the affected service.

---

## Post-Incident Checklist

- [ ] Document the root cause (OOMKill due to insufficient limits, memory leak, or node pressure)
- [ ] If memory leak was found, create a Jira ticket for the engineering team with heap profile data
- [ ] Review and update memory limits for the affected service in the Helm values or Kustomize overlays
- [ ] Verify memory alerts are firing correctly (`KubeContainerOOMKilled`, `KubeMemoryOvercommit`)
- [ ] Consider adding `GOMEMLIMIT` environment variable to all Go services for better GC behavior
- [ ] If a code fix is needed, follow [DP-003: Hotfix](../deployment/DP-003-hotfix.md) procedure
- [ ] Review resource quotas for the `clario360` namespace
- [ ] Update capacity planning with current memory growth rates
- [ ] If node pressure occurred, evaluate if the node pool needs scaling
- [ ] Update this runbook with any new patterns discovered

---

## Related Links

- [IR-006: Disk Space Exhaustion](IR-006-disk-full.md) -- Related resource exhaustion
- [SC-004: Node Pool Scaling](../scaling/SC-004-node-pool-scaling.md) -- Adding K8s nodes
- [SC-005: Capacity Planning](../scaling/SC-005-capacity-planning.md) -- Capacity planning guide
- [TS-007: High CPU Usage](../troubleshooting/TS-007-high-cpu-usage.md) -- Related resource investigation
- [DP-003: Hotfix](../deployment/DP-003-hotfix.md) -- Emergency hotfix for code-level memory leaks
- Grafana Service Health: `${GRAFANA_URL}/d/service-health`
- Grafana Cluster Overview: `${GRAFANA_URL}/d/cluster-overview`
- Go pprof documentation: https://pkg.go.dev/net/http/pprof
- Kubernetes Resource Management: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
