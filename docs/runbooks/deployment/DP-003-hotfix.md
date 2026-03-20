# DP-003: Emergency Hotfix Procedure

| Field            | Value                                            |
|------------------|--------------------------------------------------|
| **Runbook ID**   | DP-003                                           |
| **Title**        | Emergency Hotfix Procedure                       |
| **Author**       | Platform Engineering                             |
| **Last Updated** | 2026-03-08                                       |
| **Severity**     | Emergency (P1/P2)                                |
| **Services**     | Any Clario 360 platform service                  |
| **Namespace**    | clario360                                        |
| **Approvers**    | Engineering Lead (P2), VP Engineering (P1)       |
| **Est. Duration**| 30-90 minutes                                    |

---

## Summary

This runbook covers the emergency hotfix procedure for deploying a targeted fix to production when a critical bug or security vulnerability is discovered. Hotfixes bypass the normal release cycle and are deployed directly from a hotfix branch cut from the current production tag.

### When to Use This Procedure

- **P1 (Critical):** Platform is down or severely degraded, data loss risk, security breach. Skip staging. Deploy ASAP.
- **P2 (High):** Major feature broken, significant user impact. Abbreviated staging verification (15 minutes max).

### Principles

1. **Minimal change:** Only the fix. No refactoring, no feature work, no "while we're at it" changes.
2. **Traceable:** Every hotfix gets a unique branch, tag, and incident reference.
3. **Cherry-picked:** The fix must be merged back to `main` and `dev` after production is stable.

---

## Prerequisites

- [ ] Incident declared and incident channel created
- [ ] Root cause identified (at least to the file/function level)
- [ ] Engineering lead notified and verbal approval obtained
- [ ] `kubectl`, `argocd`, `gcloud`, `docker` CLI tools available
- [ ] Access to the container registry: `gcr.io/clario360-prod`
- [ ] Access to the source repository: `github.com/clario360/platform`

### Authenticate

```bash
gcloud container clusters get-credentials clario360-prod \
  --region us-central1 \
  --project clario360-prod

gcloud auth configure-docker gcr.io

argocd login argocd.clario360.internal \
  --username admin \
  --password "$(gcloud secrets versions access latest --secret=argocd-admin-password --project=clario360-prod)" \
  --grpc-web
```

---

## Procedure

### Step 1: Identify the Current Production Tag

```bash
kubectl get deployment api-gateway -n clario360 \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
# Example output: gcr.io/clario360-prod/api-gateway:v1.5.0
```

Record the current production tag:

```bash
PROD_TAG="v1.5.0"  # Replace with actual tag from above
```

### Step 2: Create Hotfix Branch from Production Tag

```bash
cd /Users/mac/clario360
git fetch --tags origin
git checkout -b hotfix/${PROD_TAG}-fix-$(date +%Y%m%d%H%M) tags/${PROD_TAG}
```

Example branch name: `hotfix/v1.5.0-fix-202603081430`

### Step 3: Apply the Minimal Fix

Make ONLY the necessary code changes to fix the issue. No additional changes.

```bash
# Example: editing a specific file
# vim internal/gateway/handler.go
# ... make the minimal fix ...
```

Verify the change is truly minimal:

```bash
git diff --stat
# Should show only 1-3 files changed with minimal lines
```

If the diff is large, reconsider whether this is really a hotfix or should go through the normal release process.

### Step 4: Run Targeted Tests

Run tests only for the affected service to minimize turnaround time:

```bash
# Replace <service-package> with the affected service's package path
# Examples:
GOWORK=off go test ./internal/gateway/... -count=1 -v -run "TestSpecificFunction"
GOWORK=off go test ./internal/iam/... -count=1 -v
GOWORK=off go test ./internal/audit/... -count=1 -v
GOWORK=off go test ./internal/workflow/... -count=1 -v
GOWORK=off go test ./internal/notification/... -count=1 -v
GOWORK=off go test ./internal/cyber/... -count=1 -v
GOWORK=off go test ./internal/data/... -count=1 -v
GOWORK=off go test ./internal/acta/... -count=1 -v
GOWORK=off go test ./internal/lex/... -count=1 -v
GOWORK=off go test ./internal/visus/... -count=1 -v
```

Run the full build to ensure no compilation errors:

```bash
GOWORK=off go build ./...
```

All tests MUST pass. If tests fail, fix them before proceeding.

### Step 5: Commit and Tag the Hotfix

```bash
git add -A
git commit -m "hotfix: <brief description of fix>

Incident: INC-XXXX
Root cause: <one-line root cause>
Impact: <one-line impact description>"

HOTFIX_TAG="${PROD_TAG}-hotfix.1"
git tag -a ${HOTFIX_TAG} -m "Hotfix for ${PROD_TAG}: <brief description>"
git push origin hotfix/${PROD_TAG}-fix-$(date +%Y%m%d%H%M)
git push origin ${HOTFIX_TAG}
```

### Step 6: Build and Push Hotfix Image

Build the Docker image for the affected service(s):

```bash
# Replace <service> with the affected service name
SERVICE="api-gateway"  # Example

docker build \
  -t gcr.io/clario360-prod/${SERVICE}:${HOTFIX_TAG} \
  -f build/docker/${SERVICE}/Dockerfile \
  --build-arg VERSION=${HOTFIX_TAG} \
  .

docker push gcr.io/clario360-prod/${SERVICE}:${HOTFIX_TAG}
```

Verify the image is in the registry:

```bash
gcloud container images list-tags gcr.io/clario360-prod/${SERVICE} \
  --filter="tags:${HOTFIX_TAG}" \
  --format="table(digest, tags, timestamp.datetime)"
```

### Step 7: Deploy via Accelerated Pipeline

#### For P1 (Skip Staging)

Update the GitOps repository directly:

```bash
cd /tmp && git clone git@github.com:clario360/deploy.git clario360-hotfix-$(date +%Y%m%d)
cd /tmp/clario360-hotfix-$(date +%Y%m%d)

sed -i '' "s|tag:.*|tag: \"${HOTFIX_TAG}\"|" gitops/clario360/charts/${SERVICE}/values-prod.yaml

git add -A
git commit -m "hotfix: deploy ${SERVICE} ${HOTFIX_TAG} to production

Incident: INC-XXXX
Skipping staging per P1 emergency procedure."

git push origin main
```

#### For P2 (Abbreviated Staging)

Deploy to staging first:

```bash
cd /tmp/clario360-hotfix-$(date +%Y%m%d)

sed -i '' "s|tag:.*|tag: \"${HOTFIX_TAG}\"|" gitops/clario360/charts/${SERVICE}/values-staging.yaml

git add -A
git commit -m "hotfix: deploy ${SERVICE} ${HOTFIX_TAG} to staging for verification"
git push origin main
```

Wait for staging sync:

```bash
argocd app sync clario360-staging --timeout 300
argocd app wait clario360-staging --timeout 300 --health --sync
```

Run a quick verification on staging (15 minutes max):

```bash
curl -s https://api.staging.clario360.com/healthz
curl -s https://api.staging.clario360.com/api/v1/health | jq .
```

Verify the specific fix works on staging, then promote to production:

```bash
sed -i '' "s|tag:.*|tag: \"${HOTFIX_TAG}\"|" gitops/clario360/charts/${SERVICE}/values-prod.yaml

git add -A
git commit -m "hotfix: promote ${SERVICE} ${HOTFIX_TAG} to production

Incident: INC-XXXX
Staging verification passed."

git push origin main
```

### Step 8: ArgoCD Sync for Production

```bash
argocd app sync clario360 --prune --force --timeout 300
argocd app wait clario360 --timeout 600 --health --sync
```

### Step 9: Verify the Fix

Verify the hotfix image is running:

```bash
kubectl get deployment ${SERVICE} -n clario360 \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
# Expected: gcr.io/clario360-prod/<service>:<hotfix-tag>
```

Verify the pod is healthy:

```bash
kubectl rollout status deployment/${SERVICE} -n clario360 --timeout=120s
```

Run smoke tests:

```bash
curl -s -o /dev/null -w "%{http_code}" https://api.clario360.com/healthz
# Expected: 200

curl -s https://api.clario360.com/api/v1/health | jq ".services.\"${SERVICE}\""
# Expected: "healthy"
```

Verify the specific issue is resolved (custom verification based on the incident):

```bash
# Example: if the fix was for a specific API endpoint
TOKEN=$(curl -s -X POST https://api.clario360.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke-test@clario360.com","password":"<SMOKE_TEST_PASSWORD>"}' \
  | jq -r '.access_token')

# Test the previously-broken endpoint
curl -s https://api.clario360.com/api/v1/<affected-endpoint> \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

### Step 10: Monitor Post-Hotfix

Monitor for 15 minutes minimum:

```bash
# Check error rates
curl -s "https://prometheus.clario360.internal/api/v1/query" \
  --data-urlencode "query=sum(rate(http_requests_total{namespace=\"clario360\",service=\"${SERVICE}\",code=~\"5..\"}[5m]))" \
  | jq '.data.result[0].value[1]'
# Expected: near 0

# Check pod restarts
kubectl get pods -n clario360 -l app.kubernetes.io/name=${SERVICE} \
  -o custom-columns="NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount"
# Expected: 0 restarts
```

### Step 11: Cherry-Pick to main and dev Branches

After the hotfix is confirmed stable in production, merge it back:

```bash
cd /Users/mac/clario360

# Cherry-pick to main
git checkout main
git pull origin main
git cherry-pick ${HOTFIX_TAG}
git push origin main

# Cherry-pick to dev
git checkout dev
git pull origin dev
git cherry-pick ${HOTFIX_TAG}
git push origin dev
```

If cherry-pick has conflicts:

```bash
# Resolve conflicts manually, then:
git add -A
git cherry-pick --continue
git push origin main  # or dev
```

### Step 12: Post-Hotfix Cleanup

Create a pull request for any cherry-pick that required conflict resolution:

```bash
git checkout -b merge-hotfix/${HOTFIX_TAG}-to-dev dev
git cherry-pick ${HOTFIX_TAG} || true
# Resolve conflicts
git add -A
git cherry-pick --continue

gh pr create \
  --repo clario360/platform \
  --title "merge: cherry-pick hotfix ${HOTFIX_TAG} to dev" \
  --body "Cherry-picking hotfix ${HOTFIX_TAG} back to dev branch. Incident: INC-XXXX" \
  --base dev
```

Update the incident ticket:

```bash
gh issue comment INC-XXXX \
  --repo clario360/platform \
  --body "Hotfix ${HOTFIX_TAG} deployed to production and cherry-picked to main/dev. Fix verified. Monitoring stable."
```

---

## Verification

| Check                          | Command                                                           | Expected                          |
|-------------------------------|-------------------------------------------------------------------|-----------------------------------|
| Hotfix image running          | `kubectl get deploy ${SERVICE} -n clario360 -o jsonpath='{.spec.template.spec.containers[0].image}'` | Shows hotfix tag |
| Pod healthy                   | `kubectl rollout status deployment/${SERVICE} -n clario360`       | Successfully rolled out           |
| Service healthy               | `curl -s https://api.clario360.com/api/v1/health \| jq`          | All services healthy              |
| Fix verified                  | Custom test for the specific issue                                | Issue no longer reproduces        |
| Cherry-picked to main         | `git log main --oneline -5`                                       | Contains hotfix commit            |
| Cherry-picked to dev          | `git log dev --oneline -5`                                        | Contains hotfix commit            |

---

## Rollback

If the hotfix makes things worse, roll back to the original production version:

```bash
cd /tmp/clario360-hotfix-*
sed -i '' "s|tag: \"${HOTFIX_TAG}\"|tag: \"${PROD_TAG}\"|" gitops/clario360/charts/${SERVICE}/values-prod.yaml

git add -A
git commit -m "rollback: revert hotfix ${HOTFIX_TAG}, restore ${PROD_TAG}

Hotfix did not resolve the issue or introduced new problems."

git push origin main

argocd app sync clario360 --prune --force --timeout 300
```

Then escalate to engineering leadership for a more thorough investigation.

---

## Related Links

- [DP-001: Deploy a New Version](./DP-001-new-release.md)
- [DP-002: Rollback to Previous Version](./DP-002-rollback.md)
- [DP-004: Run Database Migrations](./DP-004-database-migration.md)
- ArgoCD Dashboard: `https://argocd.clario360.internal`
- Incident Management: `https://incidents.clario360.internal`
- Container Registry: `https://console.cloud.google.com/gcr/images/clario360-prod`
