# IR-001: Single Service Outage

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | IR-001                                                                |
| **Title**          | Single Service Outage                                                 |
| **Severity**       | P1 -- Critical                                                        |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | api-gateway, iam-service, audit-service, workflow-engine, notification-service, cyber-service, data-service, acta-service, lex-service, visus-service |
| **Namespace**      | clario360                                                             |
| **Escalation**     | Platform Engineering Lead -> VP Engineering -> CTO                    |
| **SLA**            | Acknowledge within 5 minutes, resolve within 30 minutes               |

---

## Summary

This runbook addresses incidents where a single Clario360 platform service becomes unavailable or degraded. Scenarios include pods stuck in CrashLoopBackOff, pods not responding to health checks, pods returning HTTP errors on all requests, and pods failing to be scheduled by the Kubernetes scheduler.

---

## Symptoms

- Alerts firing for service health check failures (Prometheus `up == 0` or `probe_success == 0`).
- Upstream services returning HTTP 502/503/504 when calling the affected service.
- Kubernetes pod status showing `CrashLoopBackOff`, `Error`, `ImagePullBackOff`, `Pending`, or `OOMKilled`.
- `/healthz` or `/readyz` endpoints returning non-200 responses.
- Grafana dashboard showing zero successful requests for the service.
- Users reporting functionality outages tied to the affected service domain.

---

## Impact Assessment

| Affected Service        | Business Impact                                                      |
|-------------------------|----------------------------------------------------------------------|
| api-gateway             | **Total platform outage** -- all API traffic blocked                 |
| iam-service             | Authentication and authorization failures; all users locked out      |
| audit-service           | Audit log ingestion stops; compliance gap                            |
| workflow-engine         | All active workflows stall; task assignments stop                    |
| notification-service    | Email, SMS, push, and WebSocket notifications cease                  |
| cyber-service           | Security monitoring and threat detection halted                      |
| data-service            | Data ingestion and querying unavailable                              |
| acta-service            | Document management and compliance workflows stop                    |
| lex-service             | Legal case management and regulatory tracking offline                |
| visus-service           | Dashboards and reporting unavailable                                 |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- Access to Grafana dashboards and Prometheus/Alertmanager.
- Access to the container image registry to verify image availability.
- Permission to perform deployments and rollbacks in the `clario360` namespace.
- PagerDuty or on-call escalation access.

---

## Diagnosis Steps

Replace `<SERVICE>` with the actual service name (e.g., `api-gateway`, `iam-service`).

### Step 1: Check Pod Status

```bash
kubectl get pods -n clario360 -l app=<SERVICE> -o wide
```

Look for:
- `CrashLoopBackOff` -- container is crashing repeatedly.
- `Pending` -- pod cannot be scheduled (resource constraints or node issues).
- `ImagePullBackOff` -- image cannot be pulled from the registry.
- `OOMKilled` -- container exceeded memory limits.
- `Running` but `0/1 READY` -- readiness probe failing.

### Step 2: Describe the Pod for Events

```bash
kubectl describe pod -n clario360 -l app=<SERVICE>
```

Check the `Events` section at the bottom for:
- `FailedScheduling` -- insufficient CPU/memory on nodes.
- `FailedMount` -- secrets or configmaps not found.
- `Unhealthy` -- liveness or readiness probe failures with timestamps.
- `BackOff` -- container restart backoff details.

### Step 3: Check Current Container Logs

```bash
kubectl logs -n clario360 -l app=<SERVICE> --tail=200 --timestamps
```

### Step 4: Check Previous Container Logs (After Crash)

```bash
kubectl logs -n clario360 -l app=<SERVICE> --previous --tail=200 --timestamps
```

### Step 5: Check Resource Usage

```bash
kubectl top pod -n clario360 -l app=<SERVICE>
```

Compare CPU and memory usage against the resource limits defined in the deployment:

```bash
kubectl get deployment <SERVICE> -n clario360 -o jsonpath='{.spec.template.spec.containers[0].resources}' | jq .
```

### Step 6: Port-Forward and Test Health Endpoints Directly

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/readyz
curl -s http://localhost:8080/healthz | jq .
curl -s http://localhost:8080/readyz | jq .

kill $PF_PID
```

### Step 7: Check Deployment Rollout Status

```bash
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=30s
```

### Step 8: Check Recent Deployment History

```bash
kubectl rollout history deployment/<SERVICE> -n clario360
```

### Step 9: Check Service Endpoints

```bash
kubectl get endpoints <SERVICE> -n clario360
```

If the endpoints list is empty, no pods are passing readiness probes.

### Step 10: Check Node Health (If Pods Are Pending)

```bash
kubectl get nodes -o wide
kubectl describe node <NODE_NAME> | grep -A 10 "Conditions"
kubectl describe node <NODE_NAME> | grep -A 5 "Allocated resources"
```

---

## Resolution Steps

### Scenario A: CrashLoopBackOff

**1. Check logs for the root cause (see Diagnosis Step 3 and 4).**

**2. If caused by a bad deployment, rollback:**

```bash
kubectl rollout undo deployment/<SERVICE> -n clario360
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

**3. If caused by a missing or incorrect ConfigMap/Secret:**

```bash
kubectl get configmap -n clario360 | grep <SERVICE>
kubectl get secret -n clario360 | grep <SERVICE>
kubectl describe configmap <SERVICE>-config -n clario360
```

Fix the config and apply:

```bash
kubectl apply -f /path/to/corrected-configmap.yaml
kubectl rollout restart deployment/<SERVICE> -n clario360
```

### Scenario B: Pod Not Responding (Running but Unhealthy)

**1. Restart the deployment:**

```bash
kubectl rollout restart deployment/<SERVICE> -n clario360
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

**2. If restart does not resolve, check for resource exhaustion and increase limits:**

```bash
kubectl patch deployment <SERVICE> -n clario360 --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "1Gi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value": "1000m"}
]'
```

**3. Check if a downstream dependency is unresponsive (database, Redis, Kafka):**

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') -- wget -q -O- http://localhost:8080/readyz
```

### Scenario C: Pod Returning HTTP Errors (5xx on All Requests)

**1. Check application logs for stack traces or panic messages:**

```bash
kubectl logs -n clario360 -l app=<SERVICE> --tail=500 --timestamps | grep -i -E "panic|fatal|error|failed"
```

**2. Rollback to the last known good revision:**

```bash
kubectl rollout history deployment/<SERVICE> -n clario360
kubectl rollout undo deployment/<SERVICE> -n clario360 --to-revision=<REVISION_NUMBER>
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
```

**3. Verify the rollback resolved the issue:**

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2
curl -s -w "\n%{http_code}" http://localhost:8080/healthz
kill $PF_PID
```

### Scenario D: No Pods Scheduled (Pending)

**1. Check for resource pressure on nodes:**

```bash
kubectl describe nodes | grep -A 5 "Allocated resources"
```

**2. Scale up the node pool (GKE):**

```bash
gcloud container clusters resize clario360-cluster --node-pool=default-pool --num-nodes=<NEW_COUNT> --zone=<ZONE> --quiet
```

**3. If a PersistentVolumeClaim is pending:**

```bash
kubectl get pvc -n clario360 -l app=<SERVICE>
kubectl describe pvc -n clario360 <PVC_NAME>
```

**4. Temporarily reduce resource requests to allow scheduling:**

```bash
kubectl patch deployment <SERVICE> -n clario360 --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "128Mi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "100m"}
]'
```

### Scenario E: Scale Up for Load

```bash
kubectl scale deployment/<SERVICE> -n clario360 --replicas=5
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=120s
kubectl get pods -n clario360 -l app=<SERVICE> -o wide
```

---

## Verification

After applying a resolution, verify service health:

```bash
# 1. All pods running and ready
kubectl get pods -n clario360 -l app=<SERVICE>

# 2. Endpoints populated
kubectl get endpoints <SERVICE> -n clario360

# 3. Health check passes
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2
curl -sf http://localhost:8080/healthz && echo "HEALTHY" || echo "UNHEALTHY"
curl -sf http://localhost:8080/readyz && echo "READY" || echo "NOT READY"
kill $PF_PID

# 4. No error logs in the last 5 minutes
kubectl logs -n clario360 -l app=<SERVICE> --since=5m --timestamps | grep -i -c -E "error|panic|fatal"

# 5. Rollout complete
kubectl rollout status deployment/<SERVICE> -n clario360 --timeout=60s
```

---

## Post-Incident Checklist

- [ ] Confirm all pods are `Running` and `Ready`.
- [ ] Confirm `/healthz` and `/readyz` return HTTP 200.
- [ ] Confirm dependent services are no longer returning 502/503/504.
- [ ] Verify Prometheus alerts have cleared in Alertmanager.
- [ ] Check Grafana dashboard shows request success rate recovering.
- [ ] Notify stakeholders of resolution via incident channel.
- [ ] Create post-incident review (PIR) ticket.
- [ ] Document root cause and corrective actions.
- [ ] If rollback was performed, create a ticket to fix the forward release.
- [ ] Update deployment runbook if new failure mode was discovered.
- [ ] Review and update resource limits if OOM or CPU throttling was the cause.

---

## Related Links

| Resource                        | Link                                                        |
|---------------------------------|-------------------------------------------------------------|
| Kubernetes Troubleshooting Docs | https://kubernetes.io/docs/tasks/debug/                     |
| Grafana Dashboards              | https://grafana.clario360.internal/dashboards                |
| Alertmanager                    | https://alertmanager.clario360.internal                      |
| Deployment Pipeline             | https://github.com/clario360/platform/actions                |
| IR-002 Database Failure         | [IR-002-database-failure.md](./IR-002-database-failure.md)   |
| IR-003 Kafka Failure            | [IR-003-kafka-failure.md](./IR-003-kafka-failure.md)         |
| IR-004 Redis Failure            | [IR-004-redis-failure.md](./IR-004-redis-failure.md)         |
| IR-005 Certificate Expiry       | [IR-005-certificate-expiry.md](./IR-005-certificate-expiry.md) |
