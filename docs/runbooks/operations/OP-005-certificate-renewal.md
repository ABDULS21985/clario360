# OP-005: TLS Certificate Rotation

| Field              | Value                                      |
|--------------------|--------------------------------------------|
| **Runbook ID**     | OP-005                                     |
| **Title**          | TLS Certificate Rotation                   |
| **Frequency**      | As needed (auto-renewal monitored daily)   |
| **Estimated Time** | ~20 minutes (manual renewal)               |
| **Owner**          | Platform Team                              |
| **Last Updated**   | 2026-03-08                                 |
| **Review Cycle**   | Quarterly                                  |

## Summary

This runbook covers the management and rotation of TLS certificates for the Clario 360 platform. The platform uses cert-manager for automatic certificate lifecycle management with Let's Encrypt. This runbook documents how to verify automatic renewal, perform manual renewal when needed, verify renewed certificates are being served, and update any pinned certificates.

## Certificate Inventory

| Certificate | Secret Name | Issuer | Auto-Renew | Domains |
|-------------|-------------|--------|------------|---------|
| API Ingress | `clario360-tls` | Let's Encrypt (prod) | Yes | `api.clario360.io` |
| Grafana Ingress | `grafana-tls` | Let's Encrypt (prod) | Yes | `grafana.clario360.io` |
| Vault Ingress | `vault-tls` | Let's Encrypt (prod) | Yes | `vault.clario360.io` |
| Internal mTLS CA | `internal-ca` | Self-signed CA | Manual | Internal services |
| PostgreSQL TLS | `postgresql-tls` | Internal CA | Manual | `postgresql.clario360.svc.cluster.local` |
| Kafka TLS | `kafka-tls` | Internal CA | Manual | `*.kafka.svc.cluster.local` |
| Redis TLS | `redis-tls` | Internal CA | Manual | `redis.clario360.svc.cluster.local` |

## Prerequisites

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
```

Ensure cert-manager is installed and running:

```bash
kubectl get pods -n cert-manager
```

**Expected output:** All cert-manager pods (`cert-manager`, `cert-manager-cainjector`, `cert-manager-webhook`) are `Running` and `Ready`.

---

## Procedure 1: Check All Certificates and Expiry Dates (~5 min)

### 1a. List All cert-manager Certificates

```bash
kubectl get certificates --all-namespaces -o json | jq -r '.items[] | .metadata.namespace + "/" + .metadata.name + " | Ready: " + (.status.conditions[]? | select(.type=="Ready") | .status) + " | Expires: " + (.status.notAfter // "unknown") + " | Renewal: " + (.status.renewalTime // "unknown")'
```

**Expected output:** All certificates show `Ready: True` with expiry dates in the future.

### 1b. Check Certificates in Clario360 Namespace

```bash
kubectl get certificates -n ${NAMESPACE} -o wide
```

```bash
kubectl get certificates -n ${NAMESPACE} -o json | jq -r '.items[] | {
  name: .metadata.name,
  ready: (.status.conditions[] | select(.type=="Ready") | .status),
  notBefore: .status.notBefore,
  notAfter: .status.notAfter,
  renewalTime: .status.renewalTime,
  issuer: .spec.issuerRef.name,
  dnsNames: .spec.dnsNames
}'
```

### 1c. Check Certificates Expiring Within 30 Days

```bash
CUTOFF=$(date -u -v+30d '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+30 days' '+%Y-%m-%dT%H:%M:%SZ')
kubectl get certificates --all-namespaces -o json | jq -r --arg cutoff "${CUTOFF}" '.items[] | select(.status.notAfter != null) | select(.status.notAfter < $cutoff) | "WARNING: " + .metadata.namespace + "/" + .metadata.name + " expires " + .status.notAfter'
```

**Expected output:** No output (no certificates expiring within 30 days). Any output requires immediate action.

### 1d. Check Certificates Expiring Within 7 Days (Critical)

```bash
CUTOFF_7D=$(date -u -v+7d '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+7 days' '+%Y-%m-%dT%H:%M:%SZ')
kubectl get certificates --all-namespaces -o json | jq -r --arg cutoff "${CUTOFF_7D}" '.items[] | select(.status.notAfter != null) | select(.status.notAfter < $cutoff) | "CRITICAL: " + .metadata.namespace + "/" + .metadata.name + " expires " + .status.notAfter'
```

**Expected output:** No output. Any output is a P2 incident -- follow [IR-005](../incident-response/IR-005-certificate-expiry.md).

### 1e. Verify Certificate from TLS Secret Directly

```bash
for SECRET in clario360-tls grafana-tls vault-tls; do
  echo "=== ${SECRET} ==="
  kubectl get secret -n ${NAMESPACE} ${SECRET} -o json 2>/dev/null | jq -r '.data["tls.crt"]' | base64 -d | openssl x509 -noout -subject -dates -issuer 2>/dev/null || echo "Secret not found or not a TLS secret"
  echo ""
done
```

**Expected output:** Each certificate shows valid dates with `notAfter` well in the future.

### 1f. Check Internal CA Certificate Expiry

```bash
kubectl get secret -n ${NAMESPACE} internal-ca -o json | jq -r '.data["ca.crt"]' | base64 -d | openssl x509 -noout -subject -dates
```

**Expected output:** Internal CA certificate expiry is more than 1 year in the future (internal CAs typically have multi-year lifetimes).

### Verification

All certificates have been inventoried with their expiry dates. No certificate is expiring within 30 days.

---

## Procedure 2: cert-manager Automatic Renewal Verification (~3 min)

### 2a. Check cert-manager Controller Logs

```bash
kubectl logs -n cert-manager deploy/cert-manager --tail=50 | grep -iE "renewal|renew|certificate|error|warning"
```

**Expected output:** Log entries showing successful renewals or scheduled renewals. No persistent errors.

### 2b. Check Certificate Events

```bash
kubectl get events -n ${NAMESPACE} --sort-by=.lastTimestamp --field-selector=reason=Issuing,reason=Generated,reason=Requested | tail -20
```

```bash
kubectl get events -n ${NAMESPACE} --sort-by=.lastTimestamp | grep -iE "cert|tls|issuing|renew" | tail -20
```

**Expected output:** Recent events showing successful certificate issuance/renewal.

### 2c. Verify CertificateRequest Status

```bash
kubectl get certificaterequests -n ${NAMESPACE} -o json | jq -r '.items | sort_by(.metadata.creationTimestamp) | .[-5:] | .[] | .metadata.name + " | Approved: " + (.status.conditions[]? | select(.type=="Approved") | .status) + " | Ready: " + (.status.conditions[]? | select(.type=="Ready") | .status)'
```

**Expected output:** Recent CertificateRequests show `Approved: True` and `Ready: True`.

### 2d. Verify Issuer/ClusterIssuer Health

```bash
kubectl get issuers -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | Ready: " + (.status.conditions[]? | select(.type=="Ready") | .status) + " | Message: " + (.status.conditions[]? | select(.type=="Ready") | .message)'
```

```bash
kubectl get clusterissuers -o json | jq -r '.items[] | .metadata.name + " | Ready: " + (.status.conditions[]? | select(.type=="Ready") | .status) + " | Message: " + (.status.conditions[]? | select(.type=="Ready") | .message)'
```

**Expected output:** All issuers show `Ready: True`. ACME issuers show `The ACME account was registered with the ACME server`.

### 2e. Verify ACME Account Registration

```bash
kubectl get clusterissuers letsencrypt-prod -o json | jq -r '{
  ready: (.status.conditions[] | select(.type=="Ready") | .status),
  acme_uri: .status.acme.uri,
  last_registered: .status.acme.lastRegisteredEmail
}'
```

**Expected output:** `ready: True` with a valid ACME URI and registered email.

### Verification

cert-manager is healthy, issuers are ready, and automatic renewal is functioning.

---

## Procedure 3: Manual Certificate Renewal for Edge Cases (~10 min)

Use this procedure when:
- cert-manager auto-renewal fails
- An internal CA certificate needs rotation
- A certificate must be renewed ahead of schedule

### 3a. Force Renewal of a cert-manager Certificate

Trigger an immediate renewal by deleting the Certificate's Secret (cert-manager will re-issue):

```bash
# Replace <CERT_NAME> with the certificate name (e.g., clario360-tls)
CERT_NAME=clario360-tls

# First, verify the certificate exists
kubectl get certificate -n ${NAMESPACE} ${CERT_NAME}

# Delete the secret to trigger re-issuance
kubectl delete secret -n ${NAMESPACE} ${CERT_NAME}

# Monitor the renewal
kubectl get certificate -n ${NAMESPACE} ${CERT_NAME} -w
```

Wait for the certificate to become `Ready`:

```bash
kubectl wait --for=condition=Ready certificate/${CERT_NAME} -n ${NAMESPACE} --timeout=300s
```

**Expected output:** `certificate.cert-manager.io/<name> condition met`

### 3b. Alternative: Trigger Renewal via cmctl

```bash
kubectl exec -n cert-manager deploy/cert-manager -- cmctl renew ${CERT_NAME} -n ${NAMESPACE} 2>/dev/null || echo "cmctl not available, use secret deletion method above"
```

### 3c. Manual Certificate Generation (Internal CA)

For internal certificates not managed by cert-manager:

```bash
# Extract the CA key and certificate
kubectl get secret -n ${NAMESPACE} internal-ca -o json | jq -r '.data["ca.crt"]' | base64 -d > /tmp/ca.crt
kubectl get secret -n ${NAMESPACE} internal-ca -o json | jq -r '.data["ca.key"]' | base64 -d > /tmp/ca.key

# Generate a new private key
openssl genrsa -out /tmp/new-cert.key 4096

# Create a certificate signing request
openssl req -new -key /tmp/new-cert.key -out /tmp/new-cert.csr \
  -subj "/CN=postgresql.clario360.svc.cluster.local/O=Clario360"

# Sign the certificate with the internal CA (valid for 1 year)
openssl x509 -req -in /tmp/new-cert.csr \
  -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial \
  -out /tmp/new-cert.crt -days 365 \
  -extensions v3_req -extfile <(cat <<EOF
[v3_req]
subjectAltName = DNS:postgresql.clario360.svc.cluster.local,DNS:postgresql
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
EOF
)

# Verify the new certificate
openssl x509 -in /tmp/new-cert.crt -noout -text | head -20
```

### 3d. Update the Kubernetes Secret with the New Certificate

```bash
# Replace <SECRET_NAME> with the target secret (e.g., postgresql-tls)
SECRET_NAME=postgresql-tls

kubectl create secret tls ${SECRET_NAME} \
  -n ${NAMESPACE} \
  --cert=/tmp/new-cert.crt \
  --key=/tmp/new-cert.key \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 3e. Restart the Service to Pick Up the New Certificate

```bash
# Replace <SERVICE> with the service using this certificate
SERVICE=postgresql
kubectl rollout restart statefulset/${SERVICE} -n ${NAMESPACE}
kubectl rollout status statefulset/${SERVICE} -n ${NAMESPACE} --timeout=120s
```

### 3f. Cleanup Temporary Files

```bash
rm -f /tmp/ca.crt /tmp/ca.key /tmp/new-cert.key /tmp/new-cert.csr /tmp/new-cert.crt
```

### Verification

```bash
kubectl get certificate -n ${NAMESPACE} ${CERT_NAME} -o json | jq -r '{
  ready: (.status.conditions[] | select(.type=="Ready") | .status),
  notBefore: .status.notBefore,
  notAfter: .status.notAfter,
  renewalTime: .status.renewalTime
}'
```

---

## Procedure 4: Verify Renewed Certificate Is Served (~3 min)

### 4a. Verify Certificate via curl

```bash
curl -vI https://api.clario360.io/healthz 2>&1 | grep -E "subject:|issuer:|expire date:|start date:|SSL certificate"
```

**Expected output:** The `expire date` matches the renewed certificate's `notAfter` date.

### 4b. Verify Full Certificate Chain

```bash
curl -vI https://api.clario360.io/healthz 2>&1 | grep -A10 "Server certificate"
```

```bash
openssl s_client -connect api.clario360.io:443 -servername api.clario360.io </dev/null 2>/dev/null | openssl x509 -noout -subject -dates -issuer
```

**Expected output:**
- `subject` matches `api.clario360.io`
- `issuer` matches the expected CA (Let's Encrypt or internal CA)
- `notAfter` is the renewed expiry date

### 4c. Verify Certificate Chain Completeness

```bash
openssl s_client -connect api.clario360.io:443 -servername api.clario360.io -showcerts </dev/null 2>/dev/null | grep -E "s:|i:" | head -10
```

**Expected output:** Complete chain from leaf certificate to root CA. No `verify error` messages.

### 4d. Verify All Ingress Endpoints

```bash
for DOMAIN in api.clario360.io grafana.clario360.io vault.clario360.io; do
  echo "=== ${DOMAIN} ==="
  EXPIRY=$(openssl s_client -connect ${DOMAIN}:443 -servername ${DOMAIN} </dev/null 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null)
  echo "${DOMAIN}: ${EXPIRY}"
  echo ""
done
```

**Expected output:** All domains show valid expiry dates well in the future.

### 4e. Test Internal Service TLS (from within the cluster)

```bash
kubectl exec -n ${NAMESPACE} deploy/api-gateway -- openssl s_client -connect ${PG_HOST}:5432 -starttls postgres </dev/null 2>/dev/null | openssl x509 -noout -subject -dates 2>/dev/null || echo "PostgreSQL TLS not configured or not accessible"
```

```bash
kubectl exec -n ${NAMESPACE} deploy/api-gateway -- openssl s_client -connect redis.${NAMESPACE}.svc.cluster.local:6379 </dev/null 2>/dev/null | openssl x509 -noout -subject -dates 2>/dev/null || echo "Redis TLS not configured or not accessible"
```

### Verification

All endpoints serve the renewed certificate with the correct expiry date and a complete chain.

---

## Procedure 5: Update Any Pinned Certificates (~3 min)

Certificate pinning is used in some clients for additional security. When a certificate is rotated, pinned certificates must be updated.

### 5a. Identify Pinned Certificates

Check for certificate pins in ConfigMaps and Secrets:

```bash
kubectl get configmaps -n ${NAMESPACE} -o json | jq -r '.items[] | select(.data != null) | select(.data | to_entries[] | .value | test("ca.crt|ca-cert|tls.crt|certificate|BEGIN CERTIFICATE")) | .metadata.name'
```

### 5b. Check Service Configuration for Certificate References

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  PINS=$(kubectl get deploy -n ${NAMESPACE} ${SERVICE} -o json | jq -r '.spec.template.spec.volumes[]? | select(.secret != null or .configMap != null) | .name' | grep -iE "tls|cert|ca")
  if [ -n "${PINS}" ]; then
    echo "${SERVICE}: ${PINS}"
  fi
done
```

### 5c. Update CA Bundle in ConfigMaps

If the internal CA was rotated, update the CA bundle distributed to services:

```bash
# Extract the new CA certificate
kubectl get secret -n ${NAMESPACE} internal-ca -o json | jq -r '.data["ca.crt"]' | base64 -d > /tmp/new-ca.crt

# Update the CA bundle ConfigMap
kubectl create configmap ca-bundle \
  -n ${NAMESPACE} \
  --from-file=ca.crt=/tmp/new-ca.crt \
  --dry-run=client -o yaml | kubectl apply -f -

# Cleanup
rm -f /tmp/new-ca.crt
```

### 5d. Rolling Restart Services to Pick Up New CA Bundle

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deploy/${SERVICE} -n ${NAMESPACE}
done
```

Wait for all rollouts to complete:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout status deploy/${SERVICE} -n ${NAMESPACE} --timeout=120s
  echo "${SERVICE}: rollout complete"
done
```

### 5e. Verify Services Are Healthy After Pin Update

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  STATUS=$(kubectl exec -n ${NAMESPACE} deploy/${SERVICE} -- wget -q -O - --timeout=5 http://localhost:8080/readyz 2>/dev/null)
  EXITCODE=$?
  if [ ${EXITCODE} -eq 0 ]; then
    echo "READY: ${SERVICE}"
  else
    echo "NOT READY: ${SERVICE} - may have certificate pin issues"
  fi
done
```

**Expected output:** All 10 services return `READY`.

### Verification

All services are healthy after certificate pin updates. No TLS handshake failures in logs:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  ERRORS=$(kubectl logs -n ${NAMESPACE} deploy/${SERVICE} --since=10m 2>/dev/null | grep -ciE "tls|certificate|x509|handshake" || true)
  if [ "${ERRORS}" -gt 0 ]; then
    echo "WARNING: ${SERVICE} has ${ERRORS} TLS-related log entries in last 10 minutes"
  fi
done
```

**Expected output:** No TLS-related warnings in any service logs.

---

## Troubleshooting

### cert-manager Fails to Issue Certificate

```bash
# Check the Certificate resource status
kubectl describe certificate -n ${NAMESPACE} ${CERT_NAME}

# Check the CertificateRequest
kubectl get certificaterequests -n ${NAMESPACE} -o json | jq -r '.items[] | select(.metadata.ownerReferences[]?.name == "'${CERT_NAME}'") | .metadata.name + ": " + (.status.conditions[]? | .type + "=" + .status + " " + .message)'

# Check the Order (ACME)
kubectl get orders -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + ": " + .status.state'

# Check the Challenge (ACME)
kubectl get challenges -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | Type: " + .spec.type + " | State: " + .status.state + " | Reason: " + (.status.reason // "none")'
```

Common issues:
- **DNS challenge failed:** Verify DNS provider credentials in the Issuer
- **HTTP challenge failed:** Verify ingress is routing `/.well-known/acme-challenge/` correctly
- **Rate limited:** Let's Encrypt has rate limits; wait or use staging issuer for testing

### Certificate Not Being Served After Renewal

```bash
# Check if the ingress controller picked up the new secret
kubectl get ingress -n ${NAMESPACE} -o json | jq -r '.items[] | .metadata.name + " | TLS: " + (.spec.tls[]? | .secretName)'

# Restart the ingress controller to force reload
kubectl rollout restart deploy/ingress-nginx-controller -n ingress-nginx
kubectl rollout status deploy/ingress-nginx-controller -n ingress-nginx --timeout=120s
```

---

## Final Verification Summary

| Item | Status |
|------|--------|
| All certificates inventoried | PASS / FAIL |
| No certificates expiring within 30 days | PASS / FAIL |
| cert-manager healthy and auto-renewal active | PASS / FAIL |
| Manual renewal procedure tested (if applicable) | PASS / FAIL |
| Renewed certificate served on all endpoints | PASS / FAIL |
| Certificate pins updated (if applicable) | PASS / FAIL |
| All services healthy after rotation | PASS / FAIL |
| No TLS errors in service logs | PASS / FAIL |

## Related Runbooks

- [OP-001: Daily Checks](OP-001-daily-checks.md) (Check 6: Certificate Expiry)
- [OP-006: Secret Rotation](OP-006-secret-rotation.md)
- [IR-005: Certificate Expiry](../incident-response/IR-005-certificate-expiry.md)

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-03-08 | Platform Team | Initial version |
