# IR-004: Redis Cache Failure

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | IR-004                                                                |
| **Title**          | Redis Cache Failure                                                   |
| **Severity**       | P2 -- High                                                            |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | Redis instance in namespace `clario360`                               |
| **Namespace**      | clario360                                                             |
| **Redis Host**     | redis.clario360.svc.cluster.local                                     |
| **Default Port**   | 6379                                                                  |
| **Escalation**     | Platform Engineering Lead -> VP Engineering                           |
| **SLA**            | Acknowledge within 10 minutes, resolve within 60 minutes              |

---

## Summary

This runbook addresses incidents where the Redis cache used by Clario360 platform services becomes unavailable or degraded. Redis is used for session caching, rate limiting (api-gateway), distributed locking (workflow-engine), and general-purpose caching across services. Scenarios include Redis being unreachable, high memory usage causing evictions or OOM, slow commands degrading performance, and connection limit exhaustion.

---

## Symptoms

- Application logs showing `redis: connection refused`, `redis: LOADING Redis is loading the dataset in memory`, or `redis: OOM command not allowed`.
- api-gateway rate limiting failing open (all requests allowed) or failing closed (all requests blocked).
- Session lookups failing, causing users to be logged out unexpectedly.
- `/readyz` endpoints degraded on services that depend on Redis.
- Grafana alerts for Redis memory usage exceeding threshold.
- Grafana alerts for Redis connection count near `maxclients`.
- Increased latency across multiple services due to cache misses hitting the database.
- workflow-engine logs showing distributed lock acquisition failures.

---

## Impact Assessment

| Redis Use Case          | Affected Services                                | Business Impact                                    |
|-------------------------|--------------------------------------------------|----------------------------------------------------|
| Rate limiting           | api-gateway                                      | Rate limits fail open or closed; potential abuse or lockout |
| Session cache           | iam-service, api-gateway                         | Users logged out; increased auth latency           |
| Distributed locks       | workflow-engine                                  | Workflow step race conditions; duplicate processing |
| Response caching        | api-gateway, data-service, visus-service         | Increased DB load; higher latency                  |
| Pub/Sub (notifications) | notification-service                             | Real-time notification delivery delayed            |
| General caching         | All services                                     | Degraded performance; increased database load      |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `redis-cli` available locally or access to a pod with it.
- Redis authentication password (stored in Kubernetes secret `redis-credentials`).
- Access to Grafana dashboards for Redis metrics.

---

## Diagnosis Steps

### Step 1: Check Redis Pod Status

```bash
kubectl get pods -n clario360 -l app=redis -o wide
```

```bash
kubectl describe pod -n clario360 -l app=redis
```

### Step 2: Check Redis Pod Logs

```bash
kubectl logs -n clario360 -l app=redis --tail=200 --timestamps
```

Look for:
- `Can't handle RDB format` -- corrupted RDB file.
- `Out of memory` -- maxmemory exceeded.
- `Background saving error` -- disk full or permission issues.
- `Possible SECURITY ATTACK` -- potential abuse or misconfiguration.
- `Connection refused` from clients.

### Step 3: Test Redis Connectivity

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  PING
```

Expected output: `PONG`

### Step 4: Check Redis Memory Usage

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO memory
```

Key metrics to check:
- `used_memory_human` -- current memory usage.
- `used_memory_peak_human` -- peak memory usage.
- `maxmemory_human` -- configured maximum memory.
- `maxmemory_policy` -- eviction policy (should be `allkeys-lru` or `volatile-lru`).
- `mem_fragmentation_ratio` -- values > 1.5 indicate fragmentation.

### Step 5: Check Slow Log

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  SLOWLOG GET 25
```

### Step 6: Check Connected Clients

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO clients
```

Key metrics:
- `connected_clients` -- current client count.
- `blocked_clients` -- clients waiting on blocking operations.
- `maxclients` -- maximum allowed clients (check with `CONFIG GET maxclients`).

### Step 7: List Connected Clients with Details

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CLIENT LIST
```

### Step 8: Check Redis General Info

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO all
```

### Step 9: Check Key Count per Database

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO keyspace
```

### Step 10: Check Redis Pod Resource Usage

```bash
kubectl top pod -n clario360 -l app=redis
```

---

## Resolution Steps

### Scenario A: Redis Unreachable

**1. Check if the pod is running (see Diagnosis Step 1).**

**2. Restart the Redis pod:**

```bash
kubectl rollout restart statefulset/redis -n clario360
kubectl rollout status statefulset/redis -n clario360 --timeout=120s
```

If Redis is a standalone pod (not StatefulSet):

```bash
kubectl rollout restart deployment/redis -n clario360
kubectl rollout status deployment/redis -n clario360 --timeout=120s
```

**3. If the pod is stuck, force-delete and let the controller recreate it:**

```bash
kubectl delete pod -n clario360 -l app=redis --grace-period=0 --force
```

**4. Verify connectivity after restart:**

```bash
kubectl run redis-verify --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  PING
```

**5. Restart affected application services to re-establish connection pools:**

```bash
for svc in api-gateway iam-service workflow-engine notification-service; do
  kubectl rollout restart deployment/$svc -n clario360
done
```

### Scenario B: High Memory Usage / OOM

**1. Check current memory configuration:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG GET maxmemory
```

**2. Increase maxmemory (runtime, does not persist across restarts):**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG SET maxmemory 2gb
```

**3. Set eviction policy to LRU if not already configured:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG SET maxmemory-policy allkeys-lru
```

**4. Flush stale keys by pattern (e.g., expired session keys):**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  --scan --pattern "session:expired:*" | xargs -L 100 redis-cli -h redis.clario360.svc.cluster.local -p 6379 -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" DEL
```

**5. If a full flush is acceptable (clears all cached data -- sessions will be lost):**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  FLUSHALL ASYNC
```

**6. Update the pod memory limits to match the new maxmemory setting:**

```bash
kubectl patch deployment redis -n clario360 --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "3Gi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "2Gi"}
]'
```

### Scenario C: Slow Commands

**1. Identify slow commands (see Diagnosis Step 5).**

**2. Check if `KEYS` command is being used (should use `SCAN` instead):**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO commandstats
```

Look for `cmdstat_keys` -- if the count is high, an application is using `KEYS *`.

**3. Disable dangerous commands (runtime):**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG SET slowlog-log-slower-than 5000
```

(Logs any command taking longer than 5ms.)

**4. Reset the slow log after investigation:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  SLOWLOG RESET
```

### Scenario D: Connection Limit Reached

**1. Check current client count and maxclients:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG GET maxclients
```

**2. Increase maxclients:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG SET maxclients 20000
```

**3. Kill idle client connections older than 5 minutes:**

```bash
kubectl run redis-debug --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  CONFIG SET timeout 300
```

(Sets idle timeout to 300 seconds; Redis will close connections idle longer than this.)

**4. Review application connection pool sizes -- each service should use a bounded pool:**

```bash
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $svc ---"
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=$svc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- printenv | grep -i redis 2>/dev/null || echo "no redis env vars"
done
```

---

## Verification

```bash
# 1. Redis pod is running and ready
kubectl get pods -n clario360 -l app=redis

# 2. PING succeeds
kubectl run redis-verify --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  PING

# 3. Memory usage is below 80% of maxmemory
kubectl run redis-verify --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO memory

# 4. Connected clients below maxclients
kubectl run redis-verify --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  INFO clients

# 5. No recent slow log entries
kubectl run redis-verify --rm -it --restart=Never -n clario360 \
  --image=redis:7 \
  -- redis-cli -h redis.clario360.svc.cluster.local -p 6379 \
  -a "$(kubectl get secret redis-credentials -n clario360 -o jsonpath='{.data.password}' | base64 -d)" \
  SLOWLOG LEN

# 6. Application services pass readiness checks
for svc in api-gateway iam-service workflow-engine notification-service; do
  echo -n "$svc: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=$svc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- wget -q -O- http://localhost:8080/readyz 2>/dev/null && echo " OK" || echo " FAIL"
done
```

---

## Post-Incident Checklist

- [ ] Confirm Redis pod is `Running` and `Ready`.
- [ ] Confirm `PING` returns `PONG`.
- [ ] Confirm memory usage is below 80% of `maxmemory`.
- [ ] Confirm connected clients count is within limits.
- [ ] Confirm no new slow log entries are accumulating.
- [ ] Verify Prometheus/Grafana alerts have cleared.
- [ ] Verify application services are caching properly again.
- [ ] Check if rate limiting in api-gateway is functioning correctly.
- [ ] Notify stakeholders of resolution.
- [ ] Create post-incident review (PIR) ticket.
- [ ] Document root cause and corrective actions.
- [ ] If `FLUSHALL` was used, monitor database load as caches repopulate.
- [ ] Review and persist any runtime `CONFIG SET` changes to the Redis ConfigMap.
- [ ] Consider adding Redis Sentinel or Redis Cluster for high availability if not already configured.
- [ ] Review connection pool settings in application services.

---

## Related Links

| Resource                        | Link                                                         |
|---------------------------------|--------------------------------------------------------------|
| Redis Administration Guide      | https://redis.io/docs/management/                            |
| Redis SLOWLOG Documentation     | https://redis.io/commands/slowlog/                           |
| Grafana Redis Dashboard         | https://grafana.clario360.internal/d/redis                    |
| IR-001 Service Outage           | [IR-001-service-outage.md](./IR-001-service-outage.md)       |
| IR-002 Database Failure         | [IR-002-database-failure.md](./IR-002-database-failure.md)   |
| IR-003 Kafka Failure            | [IR-003-kafka-failure.md](./IR-003-kafka-failure.md)         |
| IR-005 Certificate Expiry       | [IR-005-certificate-expiry.md](./IR-005-certificate-expiry.md) |
