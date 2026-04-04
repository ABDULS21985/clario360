# DP-001: Deploy a New Version

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Runbook ID**   | DP-001                                             |
| **Title**        | Deploy a New Version                               |
| **Author**       | Platform Engineering                               |
| **Last Updated** | 2026-03-08                                         |
| **Severity**     | Standard Change                                    |
| **Services**     | All Clario 360 platform services                   |
| **Namespace**    | clario360                                          |
| **Approvers**    | Engineering Lead, SRE On-Call                       |
| **Est. Duration**| 45-60 minutes (including 30-minute monitoring soak)|

---

## Summary

This runbook covers the end-to-end procedure for deploying a new version of one or more Clario 360 platform services to the production GKE cluster. The deployment follows a GitOps model: image tags are updated in the Helm values files within the `deploy/gitops/clario360` repository, and ArgoCD reconciles the desired state to the cluster.

### Platform Services

| Service               | Database       | Helm Chart Path                                      |
|-----------------------|----------------|------------------------------------------------------|
| api-gateway           | --             | `deploy/gitops/clario360/charts/api-gateway`         |
| iam-service           | platform_core  | `deploy/gitops/clario360/charts/iam-service`         |
| audit-service         | platform_core  | `deploy/gitops/clario360/charts/audit-service`       |
| workflow-engine       | platform_core  | `deploy/gitops/clario360/charts/workflow-engine`     |
| notification-service  | platform_core  | `deploy/gitops/clario360/charts/notification-service`|
| cyber-service         | cyber_db       | `deploy/gitops/clario360/charts/cyber-service`       |
| data-service          | data_db        | `deploy/gitops/clario360/charts/data-service`        |
| acta-service          | acta_db        | `deploy/gitops/clario360/charts/acta-service`        |
| lex-service           | lex_db         | `deploy/gitops/clario360/charts/lex-service`         |
| visus-service         | visus_db       | `deploy/gitops/clario360/charts/visus-service`       |

---

## Prerequisites

- [ ] `kubectl` configured with production GKE cluster credentials
- [ ] `argocd` CLI authenticated to the ArgoCD server
- [ ] `git` access to the `deploy/gitops/clario360` repository
- [ ] CI pipeline has passed on all services being deployed (all tests green)
- [ ] Staging environment has been verified with the target version
- [ ] Changelog has been reviewed and approved by engineering lead
- [ ] Database migration scripts reviewed (if applicable -- see DP-004)
- [ ] Deployment window approved (avoid peak hours: 09:00-11:00 and 14:00-16:00 UTC)
- [ ] Communication sent to #deployments Slack channel

### Authenticate to GKE Cluster

```bash
gcloud container clusters get-credentials clario360-prod \
  --region us-central1 \
  --project clario360-prod
```

### Authenticate to ArgoCD

```bash
argocd login argocd.clario360.internal \
  --username admin \
  --password "$(gcloud secrets versions access latest --secret=argocd-admin-password --project=clario360-prod)" \
  --grpc-web
```

---

## Procedure

### Step 1: Record Current State

Capture the current running versions before making any changes. This provides a rollback reference.

```bash
kubectl get deployments -n clario360 \
  -o custom-columns="SERVICE:.metadata.name,IMAGE:.spec.template.spec.containers[0].image,REPLICAS:.status.readyReplicas" \
  | tee /tmp/pre-deploy-versions-$(date +%Y%m%d-%H%M%S).txt
```

Record the current ArgoCD application status:

```bash
argocd app get clario360 \
  --output json \
  | jq '.status.sync.revision' \
  | tee /tmp/pre-deploy-revision.txt
```

### Step 2: Pre-Deployment Health Check

Verify all services are healthy before starting:

```bash
kubectl get pods -n clario360 --field-selector=status.phase!=Running
```

Expected output: No resources found. If any pods are in a non-Running state, investigate before proceeding.

```bash
argocd app get clario360 --refresh
```

Verify the application status is `Synced` and `Healthy`.

### Step 3: Update Helm Values with New Image Tags

Clone the GitOps repository and create a release branch:

```bash
cd /tmp && git clone git@github.com:clario360/deploy.git clario360-deploy-$(date +%Y%m%d)
cd /tmp/clario360-deploy-$(date +%Y%m%d)
git checkout -b release/$(date +%Y%m%d-%H%M%S)
```

Update the image tag for each service being deployed. Replace `<SERVICE>` and `<NEW_TAG>` with actual values:

```bash
# For each service being deployed, update its values file:
# Example: deploying api-gateway v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/api-gateway/values-prod.yaml

# Example: deploying iam-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/iam-service/values-prod.yaml

# Example: deploying audit-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/audit-service/values-prod.yaml

# Example: deploying workflow-engine v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/workflow-engine/values-prod.yaml

# Example: deploying notification-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/notification-service/values-prod.yaml

# Example: deploying cyber-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/cyber-service/values-prod.yaml

# Example: deploying data-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/data-service/values-prod.yaml

# Example: deploying acta-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/acta-service/values-prod.yaml

# Example: deploying lex-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/lex-service/values-prod.yaml

# Example: deploying visus-service v1.5.0
sed -i '' 's|tag:.*|tag: "v1.5.0"|' gitops/clario360/charts/visus-service/values-prod.yaml
```

Commit and push:

```bash
git add -A
git commit -m "release: deploy v1.5.0 to production

Services: api-gateway, iam-service, audit-service, workflow-engine,
notification-service, cyber-service, data-service, acta-service,
lex-service, visus-service

Changelog: https://github.com/clario360/platform/blob/main/CHANGELOG.md"

git push origin release/$(date +%Y%m%d-%H%M%S)
```

Create and merge a pull request (or push directly to main if your workflow permits):

```bash
gh pr create \
  --repo clario360/deploy \
  --title "release: deploy v1.5.0 to production" \
  --body "Deploying v1.5.0. All CI checks green. Staging verified." \
  --base main

# After approval:
gh pr merge --merge --repo clario360/deploy
```

### Step 4: ArgoCD Sync

If ArgoCD auto-sync is enabled, it will detect the change within 3 minutes. To trigger a manual sync:

```bash
argocd app sync clario360 \
  --prune \
  --timeout 300
```

Monitor the sync operation:

```bash
argocd app wait clario360 \
  --timeout 600 \
  --health \
  --sync
```

### Step 5: Verify Migration Job Completion

If database migrations are included in this release, verify the migration jobs have completed:

```bash
kubectl get jobs -n clario360 -l app.kubernetes.io/component=migration \
  --sort-by='.metadata.creationTimestamp' \
  | tail -20
```

Check migration job logs for each relevant database:

```bash
# platform_core migrations (iam-service, audit-service, workflow-engine, notification-service)
kubectl logs -n clario360 job/iam-service-migration --tail=50

# cyber_db migrations
kubectl logs -n clario360 job/cyber-service-migration --tail=50

# data_db migrations
kubectl logs -n clario360 job/data-service-migration --tail=50

# acta_db migrations
kubectl logs -n clario360 job/acta-service-migration --tail=50

# lex_db migrations
kubectl logs -n clario360 job/lex-service-migration --tail=50

# visus_db migrations
kubectl logs -n clario360 job/visus-service-migration --tail=50
```

All migration jobs must show status `Completed`. If any show `Failed`, stop and refer to DP-004.

### Step 6: Verify All Pods Running New Version

```bash
kubectl get pods -n clario360 \
  -o custom-columns="NAME:.metadata.name,STATUS:.status.phase,IMAGE:.spec.containers[0].image,RESTARTS:.status.containerStatuses[0].restartCount,AGE:.metadata.creationTimestamp" \
  | sort
```

Verify:
- All pods are in `Running` state
- All pods show the new image tag
- Restart count is 0 for newly deployed pods

Check for any pods stuck in rollout:

```bash
for deploy in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "=== $deploy ==="
  kubectl rollout status deployment/$deploy -n clario360 --timeout=120s
done
```

### Step 7: Run Smoke Tests

Execute the automated smoke test suite against production:

```bash
curl -s -o /dev/null -w "%{http_code}" \
  https://api.clario360.com/healthz
# Expected: 200

curl -s -o /dev/null -w "%{http_code}" \
  https://api.clario360.com/readyz
# Expected: 200
```

Run the full smoke test suite:

```bash
curl -s https://api.clario360.com/api/v1/health | jq .
```

Expected response:

```json
{
  "status": "healthy",
  "services": {
    "api-gateway": "healthy",
    "iam-service": "healthy",
    "audit-service": "healthy",
    "workflow-engine": "healthy",
    "notification-service": "healthy",
    "cyber-service": "healthy",
    "data-service": "healthy",
    "acta-service": "healthy",
    "lex-service": "healthy",
    "visus-service": "healthy"
  }
}
```

Individual service health checks:

```bash
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "$svc: "
  kubectl exec -n clario360 deploy/$svc -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "FAILED"
done
```

### Step 8: Verify Key Functionality

**Login flow:**

```bash
curl -s -X POST https://api.clario360.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke-test@clario360.com","password":"<SMOKE_TEST_PASSWORD>"}' \
  | jq '.access_token | length > 0'
# Expected: true
```

**Dashboard data:**

```bash
TOKEN=$(curl -s -X POST https://api.clario360.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke-test@clario360.com","password":"<SMOKE_TEST_PASSWORD>"}' \
  | jq -r '.access_token')

curl -s https://api.clario360.com/api/v1/dashboard/summary \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.status'
# Expected: "ok" or 200 response
```

**Audit log write:**

```bash
curl -s https://api.clario360.com/api/v1/audit/events?limit=5 \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.total'
# Expected: numeric value > 0
```

### Step 9: Monitor for 30 Minutes Post-Deployment

Open monitoring dashboards and watch for anomalies:

```bash
# Check error rates in the last 5 minutes
kubectl logs -n clario360 -l app.kubernetes.io/part-of=clario360 \
  --since=5m --tail=100 \
  | grep -i "error\|panic\|fatal" \
  | head -20
```

Monitor key metrics via Prometheus/Grafana:

```bash
# Query error rate from Prometheus
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode 'query=sum(rate(http_requests_total{namespace="clario360",code=~"5.."}[5m])) by (service)' \
  | jq '.data.result[] | {service: .metric.service, error_rate: .value[1]}'
```

```bash
# Query request latency p99
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode 'query=histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{namespace="clario360"}[5m])) by (le, service))' \
  | jq '.data.result[] | {service: .metric.service, p99_seconds: .value[1]}'
```

```bash
# Check pod restarts in the last 30 minutes
kubectl get pods -n clario360 \
  -o custom-columns="NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount" \
  | awk '$2 > 0 {print}'
```

Set a timer and check at 10, 20, and 30 minute marks:

- [ ] 10 min: Error rate stable, no pod restarts
- [ ] 20 min: Latency within baseline, no alerts fired
- [ ] 30 min: All metrics nominal, deployment confirmed successful

### Step 10: Post-Deployment Tasks

1. Update the deployment log:

```bash
gh issue create \
  --repo clario360/platform \
  --title "Deployment: v1.5.0 to production $(date +%Y-%m-%d)" \
  --body "Deployed v1.5.0 to production at $(date -u +%H:%M:%S) UTC. All smoke tests passed. 30-minute soak period clear." \
  --label "deployment"
```

2. Notify the team:

```bash
curl -X POST https://hooks.slack.com/services/TXXXXX/BXXXXX/XXXXXXX \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"Deployment complete: v1.5.0 deployed to production. All checks passed.\"}"
```

3. Tag the production release:

```bash
cd /Users/mac/clario360
git tag -a prod-$(date +%Y%m%d) -m "Production release $(date +%Y-%m-%d)"
git push origin prod-$(date +%Y%m%d)
```

---

## Verification

| Check                          | Command                                                           | Expected                          |
|-------------------------------|-------------------------------------------------------------------|-----------------------------------|
| All pods running              | `kubectl get pods -n clario360 --field-selector=status.phase!=Running` | No resources found                |
| Correct image tags            | `kubectl get deploy -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}: {.spec.template.spec.containers[0].image}{"\n"}{end}'` | All show new tag |
| ArgoCD in sync                | `argocd app get clario360 -o json \| jq '.status.sync.status'`   | `"Synced"`                        |
| Health endpoint               | `curl -s https://api.clario360.com/healthz`                       | 200 OK                            |
| Migration jobs complete       | `kubectl get jobs -n clario360 -l app.kubernetes.io/component=migration` | All `Completed`            |
| Error rate below threshold    | Prometheus `rate(http_requests_total{code=~"5.."}[5m])`           | < 0.01                           |
| No pod restarts               | `kubectl get pods -n clario360` restart column                    | 0 for all new pods               |

---

## Rollback

If issues are detected during or after deployment, follow **DP-002: Rollback to Previous Version**.

Quick rollback command:

```bash
# Revert the GitOps commit
cd /tmp/clario360-deploy-*
git revert HEAD --no-edit
git push origin main

# Force ArgoCD sync to previous state
argocd app sync clario360 --prune --force --timeout 300
```

---

## Related Links

- [DP-002: Rollback to Previous Version](./DP-002-rollback.md)
- [DP-003: Emergency Hotfix Procedure](./DP-003-hotfix.md)
- [DP-004: Run Database Migrations](./DP-004-database-migration.md)
- [DP-005: Feature Flag Management](./DP-005-feature-flags.md)
- ArgoCD Dashboard: `https://argocd.clario360.internal`
- Grafana Dashboard: `https://grafana.clario360.internal/d/clario360-overview`
- Prometheus: `https://prometheus.clario360.internal`
- GKE Console: `https://console.cloud.google.com/kubernetes/list?project=clario360-prod`
