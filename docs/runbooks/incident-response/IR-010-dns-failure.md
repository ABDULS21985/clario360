# IR-010: DNS Resolution Failure

| Field            | Value                                    |
|------------------|------------------------------------------|
| Runbook ID       | IR-010                                   |
| Title            | DNS Resolution Failure                   |
| Severity         | P1 — Critical                            |
| Owner            | Platform Team                            |
| Last Updated     | 2026-03-08                               |
| Review Frequency | Quarterly                                |
| Approver         | Platform Lead                            |

---

## Summary

This runbook covers diagnosis and resolution of DNS resolution failures in the Clario 360 platform. DNS failures affect all inter-service communication because Kubernetes services rely on CoreDNS for service discovery. When DNS fails, services cannot resolve each other's addresses (e.g., `postgresql.clario360.svc.cluster.local`), external API calls fail, and the entire platform experiences cascading failures. This is a P1 Critical incident because DNS is a foundational dependency for every service.

---

## Symptoms

- **Alerts**: `CoreDNSDown`, `KubeDNSLatencyHigh`, `CoreDNSErrorsHigh`, `ServiceDiscoveryFailure`
- Services logging DNS resolution errors: `dial tcp: lookup <hostname>: no such host`
- Services failing to connect to PostgreSQL, Redis, or Kafka with hostname resolution errors
- Health checks failing across multiple services simultaneously
- `kubectl exec` into pods shows `nslookup` failures
- CoreDNS pods in `CrashLoopBackOff` or `Error` state
- External API integrations failing (e.g., Vault, external webhooks)
- Grafana `Service Health` dashboard (`/d/service-health`) showing all services unhealthy simultaneously
- Pod readiness probes failing across multiple deployments

---

## Impact Assessment

| Scope                        | Impact                                                          |
|------------------------------|----------------------------------------------------------------|
| CoreDNS completely down      | All service-to-service communication fails; full platform outage |
| CoreDNS degraded (partial)   | Intermittent DNS failures; sporadic service errors              |
| External DNS failure only    | Internal services work; external API calls and webhooks fail    |
| DNS cache stale/poisoned     | Services resolve to wrong IPs; traffic misrouted               |
| High DNS latency             | Increased request latency; timeouts on service calls            |
| ndots misconfiguration       | Excessive DNS queries; slow resolution for external domains     |

---

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export GRAFANA_URL=https://grafana.clario360.io
```

- `kubectl` configured with access to the `clario360` and `kube-system` namespaces
- Understanding of CoreDNS architecture in Kubernetes
- Appropriate K8s RBAC permissions (`clario360-operator` role or higher)

---

## Diagnosis Steps

### Step 1: Check CoreDNS pod status

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns -o wide
```

### Step 2: Check CoreDNS pod logs for errors

```bash
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50 --timestamps
```

### Step 3: Check CoreDNS pod resource usage

```bash
kubectl top pods -n kube-system -l k8s-app=kube-dns
```

### Step 4: Check CoreDNS pod details for crash reasons

```bash
for POD in $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[*].metadata.name}'); do
  echo "=== ${POD} ==="
  kubectl describe pod -n kube-system ${POD} | grep -A 10 "Last State:"
  echo "---"
  kubectl describe pod -n kube-system ${POD} | grep -A 5 "Events:"
done
```

### Step 5: Test DNS resolution from a debug pod

```bash
kubectl run dns-test --image=busybox:1.36 --restart=Never -n clario360 -- sleep 3600
```

```bash
kubectl exec -n clario360 dns-test -- nslookup kubernetes.default.svc.cluster.local
```

```bash
kubectl exec -n clario360 dns-test -- nslookup postgresql.clario360.svc.cluster.local
```

```bash
kubectl exec -n clario360 dns-test -- nslookup redis.clario360.svc.cluster.local
```

### Step 6: Test external DNS resolution

```bash
kubectl exec -n clario360 dns-test -- nslookup google.com
```

```bash
kubectl exec -n clario360 dns-test -- nslookup vault.clario360.io
```

### Step 7: Check /etc/resolv.conf inside a pod

```bash
kubectl exec -n clario360 dns-test -- cat /etc/resolv.conf
```

Expected output should contain:
```
nameserver <COREDNS_CLUSTER_IP>
search clario360.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

### Step 8: Verify CoreDNS service endpoint

```bash
kubectl get svc -n kube-system kube-dns -o wide
```

```bash
kubectl get endpoints -n kube-system kube-dns
```

### Step 9: Check CoreDNS ConfigMap

```bash
kubectl get configmap -n kube-system coredns -o yaml
```

### Step 10: Test DNS resolution directly against CoreDNS pod IP

```bash
COREDNS_IP=$(kubectl get pod -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].status.podIP}')
kubectl exec -n clario360 dns-test -- nslookup postgresql.clario360.svc.cluster.local ${COREDNS_IP}
```

### Step 11: Check DNS query metrics

```bash
kubectl exec -n kube-system $(kubectl get pod -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:9153/metrics | grep -E "coredns_dns_requests_total|coredns_dns_responses_total|coredns_forward_requests_total|coredns_panics_total"
```

### Step 12: Check if DNS queries are timing out (high latency)

```bash
kubectl exec -n clario360 dns-test -- sh -c "time nslookup postgresql.clario360.svc.cluster.local"
```

Expected: Resolution should complete in under 50ms.

### Step 13: Check CoreDNS deployment scaling

```bash
kubectl get deployment -n kube-system coredns -o jsonpath='{.spec.replicas}{"\t"}{.status.readyReplicas}{"\n"}'
```

### Step 14: Check if CoreDNS has enough resources

```bash
kubectl get deployment -n kube-system coredns -o jsonpath='{.spec.template.spec.containers[0].resources}' | jq .
```

### Step 15: Verify Kubernetes Service objects exist for platform services

```bash
kubectl get svc -n clario360
```

```bash
kubectl get endpoints -n clario360
```

### Step 16: Check for services with no endpoints (headless or broken)

```bash
kubectl get endpoints -n clario360 -o json | jq -r '.items[] | select(.subsets == null or (.subsets | length == 0)) | .metadata.name'
```

---

## Resolution Steps

### Option A: Restart CoreDNS pods

**Step 1**: Perform a rolling restart of CoreDNS:

```bash
kubectl rollout restart deployment/coredns -n kube-system
```

**Step 2**: Wait for the rollout to complete:

```bash
kubectl rollout status deployment/coredns -n kube-system --timeout=120s
```

**Step 3**: Verify CoreDNS pods are running:

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns -o wide
```

**Step 4**: Test DNS resolution:

```bash
kubectl exec -n clario360 dns-test -- nslookup postgresql.clario360.svc.cluster.local
```

### Option B: Fix CoreDNS ConfigMap

**Step 1**: View the current ConfigMap:

```bash
kubectl get configmap coredns -n kube-system -o yaml
```

**Step 2**: If the Corefile is corrupted or misconfigured, replace it with the correct configuration:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
            lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
            ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf {
            max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
EOF
```

**Step 3**: Restart CoreDNS to pick up the new configuration:

```bash
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout status deployment/coredns -n kube-system --timeout=120s
```

### Option C: Scale up CoreDNS (if overloaded)

**Step 1**: Check current CPU/memory usage:

```bash
kubectl top pods -n kube-system -l k8s-app=kube-dns
```

**Step 2**: Scale up CoreDNS replicas:

```bash
kubectl scale deployment coredns -n kube-system --replicas=4
```

**Step 3**: Wait for new pods to be ready:

```bash
kubectl rollout status deployment/coredns -n kube-system --timeout=120s
```

**Step 4**: Increase CoreDNS resource limits if pods are being OOMKilled:

```bash
kubectl patch deployment coredns -n kube-system --type strategic -p '{"spec":{"template":{"spec":{"containers":[{"name":"coredns","resources":{"limits":{"memory":"256Mi","cpu":"200m"},"requests":{"memory":"128Mi","cpu":"100m"}}}]}}}}'
```

### Option D: Fix resolv.conf / ndots issues

If DNS resolution is slow due to excessive search domain queries (ndots too high):

**Step 1**: Check the current ndots value:

```bash
kubectl exec -n clario360 dns-test -- cat /etc/resolv.conf
```

**Step 2**: If ndots is causing excessive queries for external domains, patch deployments to use a custom dnsConfig:

```bash
kubectl patch deployment api-gateway -n clario360 --type strategic -p '{"spec":{"template":{"spec":{"dnsConfig":{"options":[{"name":"ndots","value":"2"},{"name":"timeout","value":"2"},{"name":"attempts","value":"3"}]}}}}}'
```

**Step 3**: Apply the same patch to all services:

```bash
for SVC in iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl patch deployment ${SVC} -n clario360 --type strategic -p '{"spec":{"template":{"spec":{"dnsConfig":{"options":[{"name":"ndots","value":"2"},{"name":"timeout","value":"2"},{"name":"attempts","value":"3"}]}}}}}'
done
```

### Option E: Fix CoreDNS forward loop

If CoreDNS logs show `Loop ... detected`, the forward directive is pointing back to itself:

**Step 1**: Check the upstream DNS configuration:

```bash
kubectl debug node/$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}') -it --image=busybox -- cat /etc/resolv.conf
```

**Step 2**: Update the CoreDNS ConfigMap to use specific upstream DNS servers instead of `/etc/resolv.conf`:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
            lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
            ttl 30
        }
        prometheus :9153
        forward . 8.8.8.8 8.8.4.4 {
            max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
EOF
```

**Step 3**: Restart CoreDNS:

```bash
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout status deployment/coredns -n kube-system --timeout=120s
```

### Option F: Flush DNS cache

If DNS cache contains stale or poisoned entries:

**Step 1**: Restart CoreDNS to flush the in-memory cache:

```bash
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout status deployment/coredns -n kube-system --timeout=120s
```

**Step 2**: Restart all application pods to clear their local DNS caches:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/${SVC} -n clario360
done
```

**Step 3**: Wait for all rollouts:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout status deployment/${SVC} -n clario360 --timeout=120s
done
```

### Option G: Fix kube-dns Service (if endpoint is missing)

**Step 1**: Check if the kube-dns service has endpoints:

```bash
kubectl get endpoints -n kube-system kube-dns
```

If no endpoints are listed:

**Step 2**: Verify the selector matches the CoreDNS pods:

```bash
kubectl get svc kube-dns -n kube-system -o jsonpath='{.spec.selector}' | jq .
```

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[*].metadata.labels}' | jq .
```

**Step 3**: If labels do not match, patch the CoreDNS deployment to match:

```bash
kubectl label pods -n kube-system -l k8s-app=kube-dns --overwrite k8s-app=kube-dns
```

---

## Verification

### Step 1: Verify CoreDNS pods are running and ready

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns -o wide
```

Expected: All CoreDNS pods in `Running` state with `READY 1/1`.

### Step 2: Verify internal DNS resolution works

```bash
kubectl exec -n clario360 dns-test -- nslookup kubernetes.default.svc.cluster.local
```

```bash
kubectl exec -n clario360 dns-test -- nslookup postgresql.clario360.svc.cluster.local
```

```bash
kubectl exec -n clario360 dns-test -- nslookup redis.clario360.svc.cluster.local
```

Expected: All resolve to valid cluster IPs.

### Step 3: Verify cross-namespace DNS resolution

```bash
kubectl exec -n clario360 dns-test -- nslookup kube-dns.kube-system.svc.cluster.local
```

### Step 4: Verify external DNS resolution

```bash
kubectl exec -n clario360 dns-test -- nslookup google.com
```

```bash
kubectl exec -n clario360 dns-test -- nslookup vault.clario360.io
```

Expected: External hostnames resolve to valid IPs.

### Step 5: Verify all platform services resolve each other

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 dns-test -- nslookup ${SVC}.clario360.svc.cluster.local 2>/dev/null | grep -o "Address.*[0-9]" | tail -1 || echo "FAILED"
done
```

### Step 6: Verify all services are healthy

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

### Step 7: Verify services pass readiness checks

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/readyz 2>/dev/null || echo "NOT READY"
done
```

### Step 8: Verify DNS latency is acceptable

```bash
kubectl exec -n clario360 dns-test -- sh -c "time nslookup postgresql.clario360.svc.cluster.local"
```

Expected: Resolution completes in under 50ms.

### Step 9: Clean up the debug pod

```bash
kubectl delete pod dns-test -n clario360
```

### Step 10: Check Grafana dashboards

Open `${GRAFANA_URL}/d/service-health` and verify all services show healthy status.

---

## Post-Incident Checklist

- [ ] Document the root cause (CoreDNS crash, ConfigMap issue, resource exhaustion, loop detection, etc.)
- [ ] Verify CoreDNS alerts are correctly configured (`CoreDNSDown`, `CoreDNSErrorsHigh`)
- [ ] Review CoreDNS resource limits and adjust if pods were OOMKilled
- [ ] Confirm CoreDNS replica count is appropriate for cluster size (minimum 2 for HA)
- [ ] Review the CoreDNS Corefile for correctness
- [ ] Verify upstream DNS servers are reliable and accessible
- [ ] Check if NodeLocalDNS cache should be deployed for better performance
- [ ] Review ndots setting across all deployments (ndots:2 recommended for mixed internal/external)
- [ ] Verify pod disruption budgets (PDB) exist for CoreDNS to prevent all replicas being evicted
- [ ] Test DNS failover by deliberately killing one CoreDNS pod
- [ ] Update monitoring to detect DNS latency spikes (>100ms threshold)
- [ ] Create a Jira ticket for any infrastructure improvements needed
- [ ] Update this runbook with any new patterns discovered

---

## Related Links

- [IR-001: Service Outage](IR-001-service-outage.md) -- DNS failure causes cascading outages
- [IR-002: Database Failure](IR-002-database-failure.md) -- DNS failure prevents database connections
- [IR-004: Redis Failure](IR-004-redis-failure.md) -- DNS failure prevents Redis connections
- [TS-005: WebSocket Disconnects](../troubleshooting/TS-005-websocket-disconnects.md) -- DNS issues cause WS disconnects
- [SC-004: Node Pool Scaling](../scaling/SC-004-node-pool-scaling.md) -- More nodes may require more CoreDNS replicas
- Grafana Service Health: `${GRAFANA_URL}/d/service-health`
- CoreDNS documentation: https://coredns.io/manual/toc/
- Kubernetes DNS debugging: https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/
- NodeLocalDNS cache: https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/
