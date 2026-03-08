# IR-008: Suspected Security Incident

| Field            | Value                                    |
|------------------|------------------------------------------|
| Runbook ID       | IR-008                                   |
| Title            | Suspected Security Incident              |
| Severity         | P1 — Critical                            |
| Owner            | Platform Team / Security Team            |
| Last Updated     | 2026-03-08                               |
| Review Frequency | Quarterly                                |
| Approver         | CTO / CISO                               |

---

## Summary

This runbook covers the identification, containment, investigation, and remediation of suspected security incidents in the Clario 360 platform. Security incidents include unauthorized access, credential compromise, anomalous traffic patterns, data exfiltration attempts, and any other activity that threatens the confidentiality, integrity, or availability of the platform. This is a P1 Critical incident that requires immediate escalation and evidence preservation. All actions taken during a security incident must be documented with timestamps.

**CRITICAL**: Do NOT destroy evidence. Before taking any remediation action, complete the evidence preservation steps. All commands should be logged and timestamped.

---

## Symptoms

- **Alerts**: `HighAuthFailureRate`, `SuspiciousAPIActivity`, `AnomalousTrafficSpike`, `UnauthorizedAccessAttempt`, `WAFBlockRateHigh`
- Spike in authentication failures from a single IP or user account
- API calls to admin endpoints from non-admin users or unknown IPs
- Unusual data export or bulk read patterns (potential exfiltration)
- Service accounts used from unexpected source IPs
- JWT tokens appearing from unknown issuers or with tampered claims
- Grafana `Security Overview` (`/d/security-overview`) showing anomalous patterns
- Audit log entries showing privilege escalation or role changes not initiated by admins
- Unexpected pod deployments or configuration changes in the namespace
- Rate limiter consistently blocking a specific source
- DNS queries to known malicious domains from within the cluster

---

## Impact Assessment

| Scope                        | Impact                                                              |
|------------------------------|---------------------------------------------------------------------|
| Single user compromised      | Attacker has access to user's data and permissions                  |
| Admin account compromised    | Full platform access; data exfiltration and configuration changes   |
| Service account compromised  | Lateral movement between services; potential data access            |
| API key/secret leak          | External access to internal APIs; data exfiltration possible       |
| Database credential leak     | Direct database access; bulk data exfiltration or modification     |
| Cluster credential leak      | Full infrastructure compromise; container escape potential         |

---

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export API_URL=https://api.clario360.io
export GRAFANA_URL=https://grafana.clario360.io
export VAULT_ADDR=https://vault.clario360.io
export INCIDENT_DIR=/tmp/security-incident-$(date +%Y%m%d-%H%M%S)
mkdir -p ${INCIDENT_DIR}
```

- `kubectl` configured with access to the `clario360` namespace
- `psql` client installed (PostgreSQL 15+)
- `jq` installed for JSON processing
- `curl` installed for HTTP requests
- Access to Vault at `https://vault.clario360.io`
- Access to Grafana dashboards at `https://grafana.clario360.io`
- Appropriate K8s RBAC permissions (`clario360-admin` role)

---

## Immediate Actions (First 15 Minutes)

### Step 1: Start incident log

Create a timestamped incident log. All subsequent actions must be recorded here:

```bash
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Security incident investigation started by $(whoami)" > ${INCIDENT_DIR}/incident-log.txt
```

### Step 2: Notify the security team and escalation chain

Per the escalation matrix: P1 Critical requires notification within 15 minutes.

```
Escalation: On-call -> Platform Lead -> CTO
```

### Step 3: Do NOT restart, delete, or modify any pods or services until evidence is preserved

---

## Evidence Preservation Steps

**CRITICAL**: Complete these steps BEFORE any containment or remediation actions.

### Step 4: Capture current pod states and logs

```bash
kubectl get pods -n clario360 -o json > ${INCIDENT_DIR}/pods-state.json
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Pod states captured" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 5: Capture logs from all services

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  for POD in $(kubectl get pods -n clario360 -l app=${SVC} -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs -n clario360 ${POD} --all-containers --timestamps > ${INCIDENT_DIR}/logs-${POD}.txt 2>&1
    kubectl logs -n clario360 ${POD} --all-containers --timestamps --previous > ${INCIDENT_DIR}/logs-${POD}-previous.txt 2>&1
  done
done
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Service logs captured" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 6: Capture network policies and configurations

```bash
kubectl get networkpolicies -n clario360 -o json > ${INCIDENT_DIR}/network-policies.json
kubectl get services -n clario360 -o json > ${INCIDENT_DIR}/services.json
kubectl get ingress -n clario360 -o json > ${INCIDENT_DIR}/ingress.json
kubectl get configmaps -n clario360 -o json > ${INCIDENT_DIR}/configmaps.json
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Network config captured" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 7: Capture K8s audit logs

```bash
kubectl logs -n kube-system -l component=kube-apiserver --since=24h > ${INCIDENT_DIR}/k8s-audit-logs.txt 2>&1
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - K8s audit logs captured" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 8: Capture RBAC state

```bash
kubectl get rolebindings -n clario360 -o json > ${INCIDENT_DIR}/rolebindings.json
kubectl get clusterrolebindings -o json > ${INCIDENT_DIR}/clusterrolebindings.json
kubectl get serviceaccounts -n clario360 -o json > ${INCIDENT_DIR}/serviceaccounts.json
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - RBAC state captured" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 9: Snapshot the audit database

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- pg_dump -U ${PG_USER} -h localhost -d platform_core -t audit_logs --no-owner > ${INCIDENT_DIR}/audit_logs_dump.sql
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Audit database snapshot captured" >> ${INCIDENT_DIR}/incident-log.txt
```

---

## Diagnosis Steps

### Step 10: Check authentication failure patterns

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT
  DATE_TRUNC('hour', created_at) AS hour,
  action,
  source_ip,
  user_id,
  COUNT(*) AS failure_count
FROM audit_logs
WHERE action IN ('login_failed', 'token_rejected', 'auth_failed', 'mfa_failed')
  AND created_at > NOW() - INTERVAL '24 hours'
GROUP BY hour, action, source_ip, user_id
ORDER BY failure_count DESC
LIMIT 50;
"
```

### Step 11: Check for unauthorized privilege escalation

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT
  created_at,
  user_id,
  action,
  resource_type,
  resource_id,
  source_ip,
  details
FROM audit_logs
WHERE action IN ('role_assigned', 'role_changed', 'permission_granted', 'user_updated', 'admin_created')
  AND created_at > NOW() - INTERVAL '72 hours'
ORDER BY created_at DESC
LIMIT 50;
"
```

### Step 12: Check for anomalous API call patterns

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT
  user_id,
  source_ip,
  action,
  resource_type,
  COUNT(*) AS call_count,
  MIN(created_at) AS first_seen,
  MAX(created_at) AS last_seen
FROM audit_logs
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY user_id, source_ip, action, resource_type
HAVING COUNT(*) > 100
ORDER BY call_count DESC
LIMIT 30;
"
```

### Step 13: Identify bulk data access (potential exfiltration)

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT
  user_id,
  source_ip,
  action,
  resource_type,
  COUNT(*) AS read_count
FROM audit_logs
WHERE action IN ('read', 'list', 'export', 'download', 'bulk_read')
  AND created_at > NOW() - INTERVAL '24 hours'
GROUP BY user_id, source_ip, action, resource_type
HAVING COUNT(*) > 500
ORDER BY read_count DESC
LIMIT 20;
"
```

### Step 14: Check api-gateway access logs for suspicious patterns

```bash
kubectl logs -n clario360 -l app=api-gateway --since=24h --timestamps | grep -E "401|403|404" | awk '{print $1, $NF}' | sort | uniq -c | sort -rn | head -30 > ${INCIDENT_DIR}/gateway-errors.txt
cat ${INCIDENT_DIR}/gateway-errors.txt
```

### Step 15: Check for requests to sensitive endpoints

```bash
kubectl logs -n clario360 -l app=api-gateway --since=24h --timestamps | grep -E "/admin/|/api/v1/users|/api/v1/roles|/api/v1/tenants|/debug/" > ${INCIDENT_DIR}/sensitive-endpoint-access.txt
cat ${INCIDENT_DIR}/sensitive-endpoint-access.txt
```

### Step 16: Check for unusual network connections from pods

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  POD=$(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}')
  echo "=== ${SVC} (${POD}) ==="
  kubectl exec -n clario360 ${POD} -- wget -qO- http://localhost:8080/debug/pprof/trace?seconds=1 > /dev/null 2>&1
  kubectl exec -n clario360 ${POD} -- cat /proc/net/tcp 2>/dev/null | head -20
done > ${INCIDENT_DIR}/network-connections.txt 2>&1
```

### Step 17: Check for unexpected deployments or config changes

```bash
kubectl get events -n clario360 --sort-by='.lastTimestamp' --field-selector reason!=Pulling,reason!=Pulled,reason!=Created,reason!=Started | tail -30
```

### Step 18: Check Vault audit logs for unauthorized secret access

```bash
curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" ${VAULT_ADDR}/v1/sys/audit-hash/file -d '{"input":""}' > /dev/null 2>&1
vault audit list -format=json 2>/dev/null | jq .
```

### Step 19: Check if any service account tokens were accessed

```bash
kubectl get secrets -n clario360 -o custom-columns="NAME:.metadata.name,TYPE:.type,CREATED:.metadata.creationTimestamp" | grep service-account-token
```

### Step 20: Check DNS queries for known C2 patterns

```bash
kubectl logs -n kube-system -l k8s-app=kube-dns --since=24h | grep -iE "suspicious|malware|c2|command-and-control|exfil" > ${INCIDENT_DIR}/suspicious-dns.txt 2>&1
kubectl logs -n kube-system -l k8s-app=kube-dns --since=24h | awk '{print $NF}' | sort | uniq -c | sort -rn | head -50 > ${INCIDENT_DIR}/dns-query-frequency.txt
```

---

## Containment Steps

### Step 21: Revoke compromised user credentials

If a specific user account is compromised, disable it immediately:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
UPDATE users SET
  status = 'disabled',
  updated_at = NOW()
WHERE id = '<COMPROMISED_USER_ID>';
"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - User <COMPROMISED_USER_ID> disabled" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 22: Invalidate all sessions for the compromised user

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
DELETE FROM sessions WHERE user_id = '<COMPROMISED_USER_ID>';
"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - All sessions for <COMPROMISED_USER_ID> invalidated" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 23: Flush Redis session cache

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli -h redis KEYS "session:*<COMPROMISED_USER_ID>*" | xargs -I {} kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli -h redis DEL {}
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Redis sessions flushed for compromised user" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 24: Block suspicious IPs via NetworkPolicy

Create a deny NetworkPolicy for the suspicious source IPs:

```bash
cat <<'POLICY_EOF' | kubectl apply -n clario360 -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: block-suspicious-ips
  namespace: clario360
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - <SUSPICIOUS_IP_1>/32
        - <SUSPICIOUS_IP_2>/32
POLICY_EOF
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - NetworkPolicy applied to block suspicious IPs" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 25: Isolate a compromised service (if lateral movement suspected)

Create a NetworkPolicy to isolate the affected service from other services:

```bash
cat <<'POLICY_EOF' | kubectl apply -n clario360 -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: isolate-compromised-service
  namespace: clario360
spec:
  podSelector:
    matchLabels:
      app: <COMPROMISED_SERVICE>
  policyTypes:
  - Ingress
  - Egress
  ingress: []
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
POLICY_EOF
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Service <COMPROMISED_SERVICE> isolated via NetworkPolicy" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 26: Rotate compromised secrets in Vault

```bash
vault kv put secret/clario360/<SERVICE_NAME>/db-credentials \
  username="${PG_USER}" \
  password="$(openssl rand -base64 32)"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Database credentials rotated for <SERVICE_NAME>" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 27: Rotate JWT signing keys

```bash
vault kv put secret/clario360/iam-service/jwt-keys \
  private_key="$(openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 2>/dev/null)" \
  key_id="$(uuidgen)"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - JWT signing keys rotated" >> ${INCIDENT_DIR}/incident-log.txt
```

**Step 2**: Restart iam-service to pick up the new keys (this will invalidate ALL existing JWTs):

```bash
kubectl rollout restart deployment/iam-service -n clario360
kubectl rollout status deployment/iam-service -n clario360 --timeout=120s
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - iam-service restarted with new JWT keys" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 28: Rotate database credentials

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -c "
ALTER USER ${PG_USER} PASSWORD '$(openssl rand -base64 32)';
"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Database password rotated" >> ${INCIDENT_DIR}/incident-log.txt
```

Update the corresponding Kubernetes secrets:

```bash
kubectl create secret generic postgresql-credentials -n clario360 \
  --from-literal=username=${PG_USER} \
  --from-literal=password="<NEW_PASSWORD>" \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Step 29: Restart all services to pick up rotated credentials

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/${SVC} -n clario360
done
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - All services restarted with rotated credentials" >> ${INCIDENT_DIR}/incident-log.txt
```

Wait for all rollouts to complete:

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout status deployment/${SVC} -n clario360 --timeout=120s
done
```

---

## Resolution Steps

### Step 30: Remove isolation NetworkPolicies (after containment confirmed)

Only proceed when the security team has confirmed the threat is neutralized:

```bash
kubectl delete networkpolicy block-suspicious-ips -n clario360
kubectl delete networkpolicy isolate-compromised-service -n clario360
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Isolation NetworkPolicies removed" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 31: Re-enable compromised user account (if appropriate)

Only after password reset and MFA re-enrollment:

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
UPDATE users SET
  status = 'active',
  password_hash = NULL,
  mfa_enabled = false,
  force_password_reset = true,
  updated_at = NOW()
WHERE id = '<COMPROMISED_USER_ID>';
"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - User account re-enabled with forced password reset" >> ${INCIDENT_DIR}/incident-log.txt
```

### Step 32: Audit all admin accounts

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT
  u.id,
  u.email,
  u.status,
  u.last_login_at,
  u.mfa_enabled,
  array_agg(r.name) AS roles
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE r.name IN ('admin', 'super_admin', 'platform_admin')
GROUP BY u.id, u.email, u.status, u.last_login_at, u.mfa_enabled
ORDER BY u.last_login_at DESC;
"
```

### Step 33: Verify audit log integrity

```bash
kubectl exec -it -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U ${PG_USER} -h localhost -d platform_core -c "
SELECT COUNT(*) AS total_entries,
  COUNT(*) FILTER (WHERE entry_hash IS NULL) AS missing_hash,
  COUNT(*) FILTER (WHERE prev_hash IS NULL AND id != (SELECT MIN(id) FROM audit_logs)) AS broken_chain
FROM audit_logs
WHERE created_at > NOW() - INTERVAL '72 hours';
"
```

---

## Verification

### Step 1: Confirm all services are healthy

```bash
for SVC in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "${SVC}: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=${SVC} -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "UNHEALTHY"
done
```

### Step 2: Confirm authentication is working with new credentials

```bash
curl -s -X POST ${API_URL}/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@clario360.io","password":"<TEST_PASSWORD>"}' | jq .status
```

Expected: `"success"` or appropriate auth response.

### Step 3: Confirm suspicious IP is blocked (if NetworkPolicy still active)

```bash
kubectl get networkpolicy -n clario360
```

### Step 4: Confirm no unauthorized pods or containers

```bash
kubectl get pods -n clario360 -o custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image,STATUS:.status.phase"
```

Verify all images are from the expected container registry.

### Step 5: Check Grafana Security Overview dashboard

Open `${GRAFANA_URL}/d/security-overview` and verify:
- Auth failure rate has returned to baseline
- No anomalous traffic patterns
- Rate limiter blocks have normalized

### Step 6: Archive incident evidence

```bash
tar -czf /tmp/security-incident-$(date +%Y%m%d).tar.gz ${INCIDENT_DIR}/
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) - Evidence archived to /tmp/security-incident-$(date +%Y%m%d).tar.gz" >> ${INCIDENT_DIR}/incident-log.txt
```

---

## Post-Incident Checklist

- [ ] Complete the incident timeline document with all actions and timestamps
- [ ] Preserve all evidence in the incident directory (retain for at least 1 year)
- [ ] Conduct a post-incident review meeting within 48 hours
- [ ] Produce a Root Cause Analysis (RCA) document
- [ ] File regulatory notifications if PII/PHI data was accessed (consult legal team)
- [ ] Notify affected tenants if their data was compromised
- [ ] Verify all credentials (database, JWT, API keys, Vault tokens) have been rotated
- [ ] Confirm MFA is enforced on all admin accounts
- [ ] Review and tighten NetworkPolicies to minimize attack surface
- [ ] Review RBAC roles and remove excessive permissions
- [ ] Review API rate limiting thresholds
- [ ] Add detection rules for the attack pattern observed
- [ ] Update security monitoring alerts based on IOCs discovered
- [ ] Schedule a penetration test to validate fixes
- [ ] Update this runbook with lessons learned
- [ ] Create Jira tickets for any security hardening recommendations

---

## Related Links

- [IR-001: Service Outage](IR-001-service-outage.md) -- If the breach caused a service outage
- [IR-009: Data Corruption](IR-009-data-corruption.md) -- If data integrity was compromised
- [OP-006: Secret Rotation](../operations/OP-006-secret-rotation.md) -- Full secret rotation procedure
- [OP-010: User Management](../operations/OP-010-user-management.md) -- Admin user operations
- [TS-004: Auth Failures](../troubleshooting/TS-004-auth-failures.md) -- Authentication debugging
- [TS-009: Audit Chain Broken](../troubleshooting/TS-009-audit-chain-broken.md) -- Audit integrity verification
- Grafana Security Overview: `${GRAFANA_URL}/d/security-overview`
- Vault documentation: https://developer.hashicorp.com/vault/docs
- NIST Incident Response Guide: https://csrc.nist.gov/publications/detail/sp/800-61/rev-2/final
