# TS-007: High CPU Usage Investigation

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | TS-007                                                                |
| **Title**          | High CPU Usage Investigation                                          |
| **Severity**       | P2 -- High                                                            |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | api-gateway, iam-service, audit-service, workflow-engine, notification-service, cyber-service, data-service, acta-service, lex-service, visus-service |
| **Namespace**      | clario360                                                             |
| **Escalation**     | Platform Engineering Lead -> VP Engineering -> CTO                    |
| **SLA**            | Acknowledge within 10 minutes, resolve within 1 hour                  |

---

## Summary

This runbook addresses investigation and resolution of high CPU usage in Clario360 platform services. All services are written in Go and expose pprof profiling endpoints. High CPU can be caused by application code hotspots, excessive garbage collection pressure, tight loops, unbounded goroutine creation, expensive regular expressions, excessive logging, or external call timeouts causing goroutine buildup. CPU throttling leads to increased request latency, timeouts, and eventual pod eviction or OOMKill if the scheduler compensates by increasing memory.

---

## Symptoms

- Prometheus alert `container_cpu_usage_seconds_total` exceeding the configured threshold.
- `kubectl top pod` shows pods using close to or exceeding CPU limits.
- Increased request latency across all endpoints for the affected service.
- HTTP 504 Gateway Timeout errors from the api-gateway when proxying to the affected service.
- Pod restarts due to liveness probe timeouts caused by CPU starvation.
- Kubernetes HPA scaling events triggered by CPU utilization.
- Grafana CPU usage dashboard showing sustained spikes or a steady upward trend.

---

## Impact Assessment

| CPU Usage Level | Impact                                                               |
|-----------------|----------------------------------------------------------------------|
| 70-85% of limit | Degraded performance; increased p99 latency; HPA may trigger         |
| 85-95% of limit | Significant latency; request timeouts begin; upstream 504s likely    |
| 95-100% of limit| Service effectively unavailable; liveness probes fail; pod restarts  |
| Sustained 100%  | CPU throttling by cgroup; cascading failures to dependent services   |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `go tool pprof` installed locally (comes with Go toolchain).
- `curl` for accessing pprof HTTP endpoints.
- Access to Grafana dashboards and Prometheus.
- Familiarity with Go profiling and garbage collection diagnostics.

---

## Diagnosis Steps

Replace `<SERVICE>` with the actual service name (e.g., `api-gateway`, `workflow-engine`).

### Step 1: Identify High-CPU Pods

```bash
kubectl top pod -n clario360 --sort-by=cpu
```

Filter to a specific service:

```bash
kubectl top pod -n clario360 -l app=<SERVICE> --sort-by=cpu
```

### Step 2: Check CPU Limits and Current Usage Ratio

```bash
# Get CPU limits for the service
kubectl get deployment <SERVICE> -n clario360 -o jsonpath='{.spec.template.spec.containers[0].resources}' | jq .

# Get current CPU usage for each pod
kubectl top pod -n clario360 -l app=<SERVICE> --no-headers
```

### Step 3: Check for CPU Throttling via cgroup Metrics

```bash
POD_NAME=$(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}')

# Check throttled periods (container runtime metrics)
kubectl exec -n clario360 -it $POD_NAME -- cat /sys/fs/cgroup/cpu/cpu.stat 2>/dev/null || \
kubectl exec -n clario360 -it $POD_NAME -- cat /sys/fs/cgroup/cpu.stat 2>/dev/null
```

Look for `nr_throttled` and `throttled_time` values. High values confirm CPU throttling.

### Step 4: Check if It Is GC Pressure (Go Services)

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

# Get Go runtime metrics including GC stats
curl -s http://localhost:8080/debug/vars 2>/dev/null | jq '{
  goroutines: .goroutines,
  gc_pause_ns: .gc_pause_ns,
  mem_stats: .memstats | {Alloc, TotalAlloc, Sys, NumGC, PauseTotalNs, NumForcedGC}
}' || echo "debug/vars not available"

kill $PF_PID
```

### Step 5: Capture a CPU Profile with pprof

```bash
kubectl port-forward -n clario360 $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') 6060:6060 &
PF_PID=$!
sleep 2

# Capture a 30-second CPU profile
curl -s "http://localhost:6060/debug/pprof/profile?seconds=30" -o /tmp/<SERVICE>-cpu.prof

# Capture goroutine dump
curl -s "http://localhost:6060/debug/pprof/goroutine?debug=2" -o /tmp/<SERVICE>-goroutines.txt

# Capture heap profile
curl -s "http://localhost:6060/debug/pprof/heap" -o /tmp/<SERVICE>-heap.prof

kill $PF_PID
```

If pprof runs on the main service port (8080):

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s "http://localhost:8080/debug/pprof/profile?seconds=30" -o /tmp/<SERVICE>-cpu.prof
curl -s "http://localhost:8080/debug/pprof/goroutine?debug=2" -o /tmp/<SERVICE>-goroutines.txt
curl -s "http://localhost:8080/debug/pprof/heap" -o /tmp/<SERVICE>-heap.prof

kill $PF_PID
```

### Step 6: Analyze the CPU Profile

```bash
# Interactive analysis
go tool pprof -http=:9090 /tmp/<SERVICE>-cpu.prof

# Top CPU consumers (text mode)
go tool pprof -top /tmp/<SERVICE>-cpu.prof

# Top 20 functions by cumulative CPU time
go tool pprof -top -cum /tmp/<SERVICE>-cpu.prof | head -25

# Generate a flame graph SVG
go tool pprof -svg /tmp/<SERVICE>-cpu.prof > /tmp/<SERVICE>-flamegraph.svg
```

### Step 7: Check Goroutine Count for Goroutine Leaks

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

# Get current goroutine count
curl -s "http://localhost:8080/debug/pprof/goroutine?debug=0" | head -1

# Get detailed goroutine stack traces
curl -s "http://localhost:8080/debug/pprof/goroutine?debug=2" | head -100

# Count goroutines by state
curl -s "http://localhost:8080/debug/pprof/goroutine?debug=2" | grep "^goroutine " | wc -l

kill $PF_PID
```

A healthy service should have tens to low hundreds of goroutines. Thousands indicate a leak.

### Step 8: Analyze Goroutine Dump for Stuck Goroutines

```bash
# Count goroutines blocked on common operations
grep -c "chan receive" /tmp/<SERVICE>-goroutines.txt
grep -c "chan send" /tmp/<SERVICE>-goroutines.txt
grep -c "select" /tmp/<SERVICE>-goroutines.txt
grep -c "IO wait" /tmp/<SERVICE>-goroutines.txt
grep -c "semacquire" /tmp/<SERVICE>-goroutines.txt
grep -c "net/http" /tmp/<SERVICE>-goroutines.txt

# Find the most common goroutine stack patterns
grep "^goroutine" /tmp/<SERVICE>-goroutines.txt | head -20
```

### Step 9: Check for Excessive Logging

```bash
# Count log lines per second
kubectl logs -n clario360 -l app=<SERVICE> --since=1m --timestamps | wc -l

# Check for repeated log patterns (tight loop indicator)
kubectl logs -n clario360 -l app=<SERVICE> --tail=500 --timestamps | \
  awk '{print $2}' | sort | uniq -c | sort -rn | head -10
```

More than 1000 log lines per minute for a single pod suggests excessive logging or a tight loop.

### Step 10: Check External Call Timeouts

```bash
# Check for timeout errors in logs
kubectl logs -n clario360 -l app=<SERVICE> --tail=500 --timestamps | \
  grep -i -E "timeout|deadline exceeded|context canceled|connection refused"

# Check Prometheus metrics for external call duration
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s http://localhost:8080/metrics | grep -E "http_client_duration|external_call_duration|grpc_client_duration" | head -20

kill $PF_PID
```

### Step 11: Check Node-Level CPU

```bash
kubectl top nodes

# Check if the node running the pod is under pressure
NODE_NAME=$(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].spec.nodeName}')
kubectl describe node $NODE_NAME | grep -A 10 "Allocated resources"
kubectl describe node $NODE_NAME | grep -A 5 "Conditions"
```

### Step 12: Check HPA Status and Scaling Events

```bash
kubectl get hpa -n clario360 -l app=<SERVICE>
kubectl describe hpa <SERVICE> -n clario360
kubectl get events -n clario360 --field-selector reason=SuccessfulRescale --sort-by='.lastTimestamp' | tail -10
```

---

## Resolution Steps

### Resolution A: Increase CPU Limits (Immediate Relief)

```bash
kubectl patch deployment <SERVICE> -n clario360 --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value": "2000m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"}
]'

kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution B: Scale Out (Add More Replicas)

```bash
kubectl scale deployment/<SERVICE> -n clario360 --replicas=5
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
kubectl top pod -n clario360 -l app=<SERVICE>
```

If an HPA exists, adjust its target:

```bash
kubectl patch hpa <SERVICE> -n clario360 --type=merge -p='{"spec": {"maxReplicas": 10, "targetCPUUtilizationPercentage": 60}}'
```

### Resolution C: Fix Goroutine Leak (Code Fix Required)

If goroutine count is unbounded, identify the leak source from Step 8 analysis. Common patterns:

```bash
# Check for goroutines stuck waiting on HTTP connections without timeout
grep -B 5 "net/http.(*persistConn).readLoop" /tmp/<SERVICE>-goroutines.txt | head -20

# Check for goroutines stuck on database connections
grep -B 5 "database/sql" /tmp/<SERVICE>-goroutines.txt | head -20

# Temporary mitigation: restart the service to clear leaked goroutines
kubectl rollout restart deployment/<SERVICE> -n clario360
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution D: Reduce GC Pressure

If pprof shows high CPU in `runtime.mallocgc`, `runtime.gcBgMarkWorker`, or `runtime.scanobject`:

```bash
# Increase GOGC to reduce GC frequency (default is 100; higher = less frequent GC)
kubectl set env deployment/<SERVICE> -n clario360 GOGC=200

# Or set GOMEMLIMIT for Go 1.19+ to use soft memory limit
kubectl set env deployment/<SERVICE> -n clario360 GOMEMLIMIT=512MiB

kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution E: Add Caching to Reduce Computation

If a specific endpoint is hot, add Redis caching. Verify Redis is accessible first:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') -- \
  redis-cli -h redis.clario360.svc.cluster.local PING
```

Check current cache hit rates:

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s http://localhost:8080/metrics | grep -E "cache_hit|cache_miss"

kill $PF_PID
```

### Resolution F: Fix Excessive Logging

If logs are being emitted at an extreme rate:

```bash
# Check current log level
kubectl get deployment <SERVICE> -n clario360 -o jsonpath='{.spec.template.spec.containers[0].env}' | jq '.[] | select(.name | test("LOG"))'

# Set log level to warn to reduce log volume
kubectl set env deployment/<SERVICE> -n clario360 LOG_LEVEL=warn

kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution G: Fix External Call Timeouts Causing Goroutine Buildup

If goroutines are accumulating because external calls have no timeout or a very long timeout:

```bash
# Check service configuration for timeout values
kubectl get configmap <SERVICE>-config -n clario360 -o yaml | grep -i timeout

# Set explicit timeout environment variables
kubectl set env deployment/<SERVICE> -n clario360 \
  HTTP_CLIENT_TIMEOUT=10s \
  DB_QUERY_TIMEOUT=30s \
  KAFKA_PUBLISH_TIMEOUT=5s

kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

### Resolution H: Rollback If CPU Spike Correlates with Recent Deployment

```bash
# Check deployment history
kubectl rollout history deployment/<SERVICE> -n clario360

# Rollback to previous revision
kubectl rollout undo deployment/<SERVICE> -n clario360
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s

# Verify CPU has dropped
sleep 30
kubectl top pod -n clario360 -l app=<SERVICE>
```

---

## Verification

After applying a resolution, verify CPU has returned to normal:

```bash
# 1. Check CPU usage has decreased
kubectl top pod -n clario360 -l app=<SERVICE> --no-headers

# 2. Wait 2 minutes and check again for stability
sleep 120
kubectl top pod -n clario360 -l app=<SERVICE> --no-headers

# 3. Verify request latency has returned to normal
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

# Check p99 latency from metrics
curl -s http://localhost:8080/metrics | grep -E "http_request_duration_seconds" | grep "quantile=\"0.99\""

# Health check should respond quickly (under 100ms)
time curl -sf http://localhost:8080/healthz > /dev/null
time curl -sf http://localhost:8080/readyz > /dev/null

kill $PF_PID

# 4. Check goroutine count is reasonable
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2
curl -s "http://localhost:8080/debug/pprof/goroutine?debug=0" | head -1
kill $PF_PID

# 5. Check no CPU throttling alerts in Prometheus
kubectl port-forward -n clario360 svc/prometheus 9090:9090 &
PF_PID=$!
sleep 2
curl -s "http://localhost:9090/api/v1/alerts" | jq '.data.alerts[] | select(.labels.alertname | test("cpu|CPU"))'
kill $PF_PID

# 6. Verify no recent pod restarts
kubectl get pods -n clario360 -l app=<SERVICE> -o custom-columns=NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount,STATUS:.status.phase

# 7. Check logs for errors in last 5 minutes
kubectl logs -n clario360 -l app=<SERVICE> --since=5m --timestamps | grep -i -c -E "error|panic|fatal"
```

---

## Post-Incident Checklist

- [ ] Confirm CPU usage is below 70% of limits for all pods.
- [ ] Confirm request latency has returned to baseline.
- [ ] Confirm no CPU throttling is occurring (check cgroup stats).
- [ ] Confirm goroutine count is stable and not growing.
- [ ] Verify Prometheus CPU alerts have cleared in Alertmanager.
- [ ] If code fix was applied, create a ticket for proper review and merge.
- [ ] If CPU limits were increased, review and right-size after the incident.
- [ ] If GOGC was changed, monitor memory usage for potential increase.
- [ ] Save pprof profiles in the incident archive for post-incident analysis.
- [ ] Document root cause and corrective actions.
- [ ] Review HPA configuration if scaling was inadequate.
- [ ] Consider adding CPU usage rate-of-change alerts for earlier detection.

---

## Related Links

| Resource                         | Link                                                                     |
|----------------------------------|--------------------------------------------------------------------------|
| Go pprof Documentation           | https://pkg.go.dev/net/http/pprof                                        |
| Go GC Guide                      | https://tip.golang.org/doc/gc-guide                                      |
| Kubernetes CPU Management        | https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/ |
| Grafana Dashboards               | https://grafana.clario360.internal/dashboards                            |
| Alertmanager                     | https://alertmanager.clario360.internal                                  |
| IR-001 Service Outage            | [../incident-response/IR-001-service-outage.md](../incident-response/IR-001-service-outage.md) |
| TS-008 Connection Pool           | [TS-008-connection-pool-exhaustion.md](./TS-008-connection-pool-exhaustion.md) |
