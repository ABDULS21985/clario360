# IR-005: TLS Certificate Expiration

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | IR-005                                                                |
| **Title**          | TLS Certificate Expiration                                            |
| **Severity**       | P2 -- High                                                            |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | All Clario360 services with TLS endpoints, Ingress, inter-service mTLS |
| **Namespace**      | clario360                                                             |
| **Cert Manager**   | cert-manager namespace                                                |
| **Escalation**     | Platform Engineering Lead -> Security Engineering Lead -> VP Engineering |
| **SLA**            | Acknowledge within 10 minutes, resolve within 60 minutes              |

---

## Summary

This runbook addresses incidents related to TLS certificate expiration or renewal failure in the Clario360 platform. Scenarios include cert-manager failing to automatically renew certificates, certificates that have already expired causing connection failures, and CA trust chain issues preventing services from establishing secure connections. Expired certificates can cause total service outages, API failures, and broken inter-service communication.

---

## Symptoms

- Users seeing browser warnings "Your connection is not private" or "NET::ERR_CERT_DATE_INVALID".
- Application logs showing `x509: certificate has expired or is not yet valid`.
- Inter-service calls failing with `tls: failed to verify certificate` or `x509: certificate signed by unknown authority`.
- Ingress controller returning HTTP 503 or dropping HTTPS connections.
- cert-manager Certificate resources showing `Ready: False` or `Issuing` status stuck.
- Grafana alerts for certificate expiry within N days.
- Prometheus `certmanager_certificate_expiration_timestamp_seconds` metric showing certificates near or past expiry.
- Webhook calls (admission controllers, API callbacks) failing with TLS errors.

---

## Impact Assessment

| Certificate Scope            | Affected Components                          | Business Impact                                    |
|------------------------------|----------------------------------------------|----------------------------------------------------|
| Ingress TLS (wildcard)       | All external HTTPS traffic                   | **Total external outage** -- all users affected    |
| Inter-service mTLS           | Service-to-service communication             | Internal service calls fail; cascading failures    |
| api-gateway TLS              | api-gateway ingress                          | All API requests fail                              |
| PostgreSQL TLS               | Database connections                         | Services cannot connect to databases               |
| Kafka TLS                    | Kafka broker and client connections          | Event streaming stops                              |
| Redis TLS (if enabled)       | Redis client connections                     | Caching and rate limiting fail                     |
| Webhook TLS                  | Admission controllers, cert-manager webhooks | Deployments and cert renewals blocked              |

---

## Prerequisites

- `kubectl` configured with cluster access and permissions on `clario360` and `cert-manager` namespaces.
- `openssl` CLI installed locally.
- Access to the DNS provider for domain validation challenges (if using DNS01).
- Access to the CA (Let's Encrypt, internal CA) credentials and configuration.
- cert-manager ClusterIssuer or Issuer configuration documentation.

---

## Diagnosis Steps

### Step 1: List All Certificates and Their Status

```bash
kubectl get certificates -n clario360 -o wide
```

```bash
kubectl get certificates --all-namespaces -o wide
```

Look for certificates where `READY` is `False` or `EXPIRY` is in the past or within the next 24 hours.

### Step 2: Describe a Specific Certificate for Detailed Status

```bash
kubectl describe certificate <CERTIFICATE_NAME> -n clario360
```

Check the `Status` section for:
- `Conditions` -- `Ready` condition message.
- `Not After` -- expiry timestamp.
- `Renewal Time` -- when cert-manager will attempt renewal.
- `Events` -- recent cert-manager actions and errors.

### Step 3: Check CertificateRequest Status

```bash
kubectl get certificaterequests -n clario360
```

```bash
kubectl describe certificaterequest <REQUEST_NAME> -n clario360
```

### Step 4: Check cert-manager Controller Logs

```bash
kubectl logs -n cert-manager -l app=cert-manager --tail=300 --timestamps
```

Look for:
- `error preparing certificate` -- configuration issue.
- `failed to perform self-check` -- HTTP01 challenge cannot be reached.
- `error getting keypair` -- secret access issue.
- `ACME server error` -- Let's Encrypt rate limit or API issue.
- `no matching issuer` -- Issuer/ClusterIssuer not found.

### Step 5: Check cert-manager Webhook Logs

```bash
kubectl logs -n cert-manager -l app=cert-manager-webhook --tail=100 --timestamps
```

### Step 6: Check Issuer/ClusterIssuer Status

```bash
kubectl get clusterissuers -o wide
kubectl describe clusterissuer <ISSUER_NAME>
```

```bash
kubectl get issuers -n clario360 -o wide
kubectl describe issuer <ISSUER_NAME> -n clario360
```

### Step 7: Check the Actual Certificate Content in the Secret

```bash
kubectl get secret <TLS_SECRET_NAME> -n clario360 -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates -subject -issuer
```

This shows:
- `notBefore` -- when the certificate became valid.
- `notAfter` -- when the certificate expires.
- `subject` -- the domain(s) covered.
- `issuer` -- the CA that signed it.

### Step 8: Check Certificate Chain Validity

```bash
kubectl get secret <TLS_SECRET_NAME> -n clario360 -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -text | grep -A 2 "Validity"
```

### Step 9: Test TLS Connection Externally

```bash
openssl s_client -connect <SERVICE_DOMAIN>:443 -servername <SERVICE_DOMAIN> </dev/null 2>/dev/null | openssl x509 -noout -dates -subject
```

For example:

```bash
openssl s_client -connect api.clario360.com:443 -servername api.clario360.com </dev/null 2>/dev/null | openssl x509 -noout -dates -subject
```

### Step 10: Check Certificate Expiry from Inside the Cluster

```bash
kubectl run tls-debug --rm -it --restart=Never -n clario360 \
  --image=alpine/openssl:latest \
  -- s_client -connect api-gateway.clario360.svc.cluster.local:8080 -servername api-gateway </dev/null 2>/dev/null | openssl x509 -noout -dates
```

### Step 11: Check All Certificates Expiring Within 7 Days

```bash
for secret in $(kubectl get secrets -n clario360 --field-selector type=kubernetes.io/tls -o jsonpath='{.items[*].metadata.name}'); do
  EXPIRY=$(kubectl get secret "$secret" -n clario360 -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
  if [ -n "$EXPIRY" ]; then
    EXPIRY_EPOCH=$(date -j -f "%b %d %T %Y %Z" "$EXPIRY" "+%s" 2>/dev/null || date -d "$EXPIRY" "+%s" 2>/dev/null)
    NOW_EPOCH=$(date "+%s")
    DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))
    if [ "$DAYS_LEFT" -lt 7 ]; then
      echo "WARNING: $secret expires in $DAYS_LEFT days ($EXPIRY)"
    fi
  fi
done
```

### Step 12: Check ACME Challenge Status (If Using Let's Encrypt)

```bash
kubectl get challenges -n clario360
kubectl describe challenge <CHALLENGE_NAME> -n clario360
```

```bash
kubectl get orders -n clario360
kubectl describe order <ORDER_NAME> -n clario360
```

---

## Resolution Steps

### Scenario A: cert-manager Renewal Failure

**1. Check cert-manager pod status and restart if needed:**

```bash
kubectl get pods -n cert-manager
kubectl rollout restart deployment/cert-manager -n cert-manager
kubectl rollout restart deployment/cert-manager-webhook -n cert-manager
kubectl rollout restart deployment/cert-manager-cainjector -n cert-manager
kubectl rollout status deployment/cert-manager -n cert-manager --timeout=120s
```

**2. If the Issuer/ClusterIssuer is in an error state, check and fix its configuration:**

```bash
kubectl describe clusterissuer letsencrypt-prod
```

If the ACME account key is corrupted, delete and let it re-register:

```bash
kubectl delete secret letsencrypt-prod-account-key -n cert-manager
kubectl rollout restart deployment/cert-manager -n cert-manager
```

**3. If HTTP01 challenge is failing (challenge pod cannot be reached):**

```bash
# Check if the challenge solver pod is running
kubectl get pods -n clario360 -l acme.cert-manager.io/http01-solver=true

# Check ingress for the challenge
kubectl get ingress -n clario360 -l acme.cert-manager.io/http01-solver=true
```

Ensure the ingress controller is routing `/.well-known/acme-challenge/` correctly.

**4. If DNS01 challenge is failing:**

```bash
kubectl logs -n cert-manager -l app=cert-manager --tail=100 | grep -i dns
```

Verify DNS provider credentials:

```bash
kubectl get secret <DNS_PROVIDER_SECRET> -n cert-manager
```

### Scenario B: Certificate Already Expired

**1. Delete the expired certificate resource to trigger re-issuance:**

```bash
kubectl delete certificate <CERTIFICATE_NAME> -n clario360
```

Then re-apply the Certificate manifest:

```bash
kubectl apply -f /path/to/certificate-manifest.yaml
```

**2. If the certificate manifest is not available, create a new Certificate resource:**

```bash
kubectl apply -n clario360 -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: clario360-tls
  namespace: clario360
spec:
  secretName: clario360-tls-secret
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - "api.clario360.com"
    - "*.clario360.com"
  duration: 2160h    # 90 days
  renewBefore: 720h  # 30 days before expiry
EOF
```

**3. Monitor the certificate issuance:**

```bash
kubectl get certificate clario360-tls -n clario360 -w
```

Wait until `READY` shows `True`.

**4. Force the ingress controller to pick up the new certificate:**

```bash
kubectl rollout restart deployment/ingress-nginx-controller -n ingress-nginx
```

### Scenario C: Force Certificate Renewal

**1. Trigger a renewal by deleting the TLS secret (cert-manager will detect and re-issue):**

```bash
kubectl delete secret <TLS_SECRET_NAME> -n clario360
```

**2. Alternatively, use the cmctl CLI to force renewal:**

```bash
kubectl cert-manager renew <CERTIFICATE_NAME> -n clario360
```

Or if `cmctl` is installed:

```bash
cmctl renew <CERTIFICATE_NAME> -n clario360
```

**3. Monitor the renewal process:**

```bash
kubectl get certificaterequests -n clario360 -w
kubectl get certificate <CERTIFICATE_NAME> -n clario360 -w
```

### Scenario D: CA Trust Issues (Internal CA)

**1. Check if the CA certificate bundle is present and valid:**

```bash
kubectl get configmap ca-certificates -n clario360 -o jsonpath='{.data.ca\.crt}' | openssl x509 -noout -dates -subject
```

**2. If the CA cert has expired, update the ConfigMap with the new CA cert:**

```bash
kubectl create configmap ca-certificates -n clario360 \
  --from-file=ca.crt=/path/to/new-ca-certificate.pem \
  --dry-run=client -o yaml | kubectl apply -f -
```

**3. Restart all services to pick up the new CA bundle:**

```bash
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/$svc -n clario360
done
```

**4. If services use a mounted CA bundle volume, verify it is updated:**

```bash
kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=api-gateway -o jsonpath='{.items[0].metadata.name}') -- cat /etc/ssl/certs/ca-certificates.crt | openssl x509 -noout -dates -subject
```

### Scenario E: Webhook Certificate Expired (cert-manager Bootstrap Issue)

If cert-manager's own webhook certificate has expired, cert-manager cannot renew any certificates (circular dependency).

**1. Delete the cert-manager webhook secret to force regeneration:**

```bash
kubectl delete secret cert-manager-webhook-ca -n cert-manager
kubectl delete secret cert-manager-webhook-tls -n cert-manager
```

**2. Restart all cert-manager components:**

```bash
kubectl rollout restart deployment/cert-manager -n cert-manager
kubectl rollout restart deployment/cert-manager-webhook -n cert-manager
kubectl rollout restart deployment/cert-manager-cainjector -n cert-manager
```

**3. Wait for cert-manager to come back healthy:**

```bash
kubectl rollout status deployment/cert-manager -n cert-manager --timeout=120s
kubectl rollout status deployment/cert-manager-webhook -n cert-manager --timeout=120s
kubectl rollout status deployment/cert-manager-cainjector -n cert-manager --timeout=120s
```

**4. Verify the webhook is operational:**

```bash
kubectl get validatingwebhookconfiguration cert-manager-webhook -o yaml | grep -A 3 caBundle
```

---

## Verification

```bash
# 1. All certificates show Ready: True
kubectl get certificates -n clario360 -o wide

# 2. Certificate secret contains a valid, non-expired cert
kubectl get secret <TLS_SECRET_NAME> -n clario360 -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates -subject -issuer

# 3. External TLS connectivity works
openssl s_client -connect api.clario360.com:443 -servername api.clario360.com </dev/null 2>/dev/null | openssl x509 -noout -dates

# 4. No ACME challenges stuck
kubectl get challenges --all-namespaces

# 5. cert-manager pods are healthy
kubectl get pods -n cert-manager

# 6. cert-manager logs show no errors
kubectl logs -n cert-manager -l app=cert-manager --tail=50 --timestamps | grep -i error

# 7. All Clario360 services pass health checks
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "$svc: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=$svc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- wget -q -O- http://localhost:8080/healthz 2>/dev/null && echo " OK" || echo " FAIL"
done

# 8. Ingress is serving HTTPS correctly
curl -sI https://api.clario360.com/healthz -o /dev/null -w "%{http_code}" 2>/dev/null
```

---

## Post-Incident Checklist

- [ ] Confirm all Certificate resources show `Ready: True`.
- [ ] Confirm renewed certificates have correct `notAfter` dates (90 days out for Let's Encrypt).
- [ ] Confirm external HTTPS access is working for all domains.
- [ ] Confirm inter-service mTLS (if enabled) is functioning.
- [ ] Verify Prometheus/Grafana alerts for certificate expiry have cleared.
- [ ] Verify cert-manager is healthy and actively monitoring certificates.
- [ ] Notify stakeholders of resolution.
- [ ] Create post-incident review (PIR) ticket.
- [ ] Document root cause and corrective actions.
- [ ] Set up proactive alerting for certificates expiring within 14 days.
- [ ] Review `renewBefore` setting on all Certificate resources (should be at least 720h / 30 days).
- [ ] Verify backup ClusterIssuer exists (e.g., letsencrypt-staging for testing).
- [ ] If CA trust was the issue, distribute the new CA bundle to all environments.
- [ ] Consider implementing certificate monitoring with a tool like `cert-exporter`.

---

## Related Links

| Resource                        | Link                                                         |
|---------------------------------|--------------------------------------------------------------|
| cert-manager Documentation      | https://cert-manager.io/docs/                                |
| Let's Encrypt Rate Limits       | https://letsencrypt.org/docs/rate-limits/                    |
| OpenSSL Quick Reference         | https://www.openssl.org/docs/man3.0/man1/                    |
| Grafana Cert Dashboard          | https://grafana.clario360.internal/d/certificates             |
| IR-001 Service Outage           | [IR-001-service-outage.md](./IR-001-service-outage.md)       |
| IR-002 Database Failure         | [IR-002-database-failure.md](./IR-002-database-failure.md)   |
| IR-003 Kafka Failure            | [IR-003-kafka-failure.md](./IR-003-kafka-failure.md)         |
| IR-004 Redis Failure            | [IR-004-redis-failure.md](./IR-004-redis-failure.md)         |
