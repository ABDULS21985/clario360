# DP-002: Rollback to Previous Version

| Field            | Value                                            |
|------------------|--------------------------------------------------|
| **Runbook ID**   | DP-002                                           |
| **Title**        | Rollback to Previous Version                     |
| **Author**       | Platform Engineering                             |
| **Last Updated** | 2026-03-08                                       |
| **Severity**     | Emergency / High Priority                        |
| **Services**     | All Clario 360 platform services                 |
| **Namespace**    | clario360                                        |
| **Approvers**    | SRE On-Call (verbal approval sufficient for P1)  |
| **Est. Duration**| 5-15 minutes (rollback), up to 60 minutes (with DB rollback) |

---

## Summary

This runbook covers rolling back one or more Clario 360 platform services to their previous known-good version. The primary rollback mechanism is a GitOps revert: reverting the Helm values commit in the `deploy/gitops/clario360` repository and forcing an ArgoCD sync. If database migrations were part of the failed release, an additional database migration rollback step is required.

### When to Rollback

- Service error rate exceeds 5% after deployment
- P99 latency exceeds 2x baseline after deployment
- Critical functionality broken (login, dashboard, data ingestion)
- Repeated pod crashes or OOMKills
- Data integrity issues detected
- Security vulnerability discovered in the deployed version

### Decision Matrix

| Scenario                                | Action                                   |
|-----------------------------------------|------------------------------------------|
| Single service failure, no DB migration | Rollback single service (Step 2a)        |
| Single service failure, with DB migration| Rollback service + DB migration (Steps 2a + 3) |
| Multiple service failures               | Full rollback (Step 2b)                  |
| Full platform outage                    | Full rollback + incident escalation      |

---

## Prerequisites

- [ ] `kubectl` configured with production GKE cluster credentials
- [ ] `argocd` CLI authenticated to the ArgoCD server
- [ ] `git` access to the `deploy/gitops/clario360` repository
- [ ] Previous good version identified (from pre-deployment snapshot or git history)
- [ ] Incident channel created (if triggered by an incident)

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

### Step 1: Identify the Commit to Revert

Find the deployment commit that introduced the problem:

```bash
cd /tmp && git clone git@github.com:clario360/deploy.git clario360-rollback-$(date +%Y%m%d)
cd /tmp/clario360-rollback-$(date +%Y%m%d)
git log --oneline -10
```

Example output:

```
a1b2c3d release: deploy v1.5.0 to production
e4f5g6h release: deploy v1.4.0 to production
...
```

Identify the commit hash of the bad deployment (e.g., `a1b2c3d`).

Verify the previous good version from the pre-deployment snapshot:

```bash
cat /tmp/pre-deploy-versions-*.txt
```

Or query the cluster directly for the previous ReplicaSet:

```bash
kubectl rollout history deployment/api-gateway -n clario360
```

### Step 2a: Rollback a Single Service via GitOps Revert (< 5 minutes)

If only one service needs to be rolled back, manually revert the image tag:

```bash
cd /tmp/clario360-rollback-$(date +%Y%m%d)
git checkout main && git pull origin main

# Revert the specific service's image tag to the previous version
# Example: rolling back api-gateway from v1.5.0 to v1.4.0
sed -i '' 's|tag: "v1.5.0"|tag: "v1.4.0"|' gitops/clario360/charts/api-gateway/values-prod.yaml

git add -A
git commit -m "rollback: revert api-gateway to v1.4.0

Reason: <brief description of issue>
Incident: INC-XXXX"

git push origin main
```

### Step 2b: Full Rollback via GitOps Revert (< 5 minutes)

Revert the entire deployment commit:

```bash
cd /tmp/clario360-rollback-$(date +%Y%m%d)
git checkout main && git pull origin main

# Revert the deployment commit
git revert a1b2c3d --no-edit

git push origin main
```

### Step 3: ArgoCD Force Sync

Trigger ArgoCD to immediately apply the reverted state:

```bash
argocd app sync clario360 \
  --prune \
  --force \
  --timeout 300
```

Monitor the rollback progress:

```bash
argocd app wait clario360 \
  --timeout 600 \
  --health \
  --sync
```

Verify ArgoCD shows `Synced` and `Healthy`:

```bash
argocd app get clario360 -o json | jq '{sync: .status.sync.status, health: .status.health.status}'
```

If ArgoCD sync is stuck, force a hard refresh:

```bash
argocd app get clario360 --hard-refresh
argocd app sync clario360 --force --prune --timeout 300
```

### Step 4: Verify Rollback of Pods

```bash
kubectl get pods -n clario360 \
  -o custom-columns="NAME:.metadata.name,STATUS:.status.phase,IMAGE:.spec.containers[0].image,RESTARTS:.status.containerStatuses[0].restartCount" \
  | sort
```

Verify all pods show the previous (good) image tag.

Watch the rollout complete:

```bash
for deploy in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "=== $deploy ==="
  kubectl rollout status deployment/$deploy -n clario360 --timeout=120s
done
```

### Step 5: Database Migration Rollback (If Applicable)

**WARNING:** Only perform this step if database migrations were part of the failed release AND the migrations are causing data issues. Many migrations are backwards-compatible and do not require rollback.

Check if migrations were applied:

```bash
kubectl get jobs -n clario360 -l app.kubernetes.io/component=migration \
  --sort-by='.metadata.creationTimestamp' \
  | tail -10
```

If migration rollback is needed, run the down migration for each affected database:

#### platform_core (iam-service, audit-service, workflow-engine, notification-service)

```bash
kubectl run migration-rollback-platform-core \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/iam-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 platform-core-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

#### cyber_db

```bash
kubectl run migration-rollback-cyber-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/cyber-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 cyber-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

#### data_db

```bash
kubectl run migration-rollback-data-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/data-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 data-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

#### acta_db

```bash
kubectl run migration-rollback-acta-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/acta-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 acta-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

#### lex_db

```bash
kubectl run migration-rollback-lex-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/lex-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 lex-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

#### visus_db

```bash
kubectl run migration-rollback-visus-db \
  --namespace=clario360 \
  --image=gcr.io/clario360-prod/visus-service:v1.4.0 \
  --restart=Never \
  --rm -it \
  --env="DATABASE_URL=$(kubectl get secret -n clario360 visus-db-credentials -o jsonpath='{.data.url}' | base64 -d)" \
  -- go run cmd/migrator/main.go --direction=down --steps=1
```

Verify the migration rollback completed successfully by checking the migration version:

```bash
kubectl exec -n clario360 deploy/iam-service -- go run cmd/migrator/main.go --version
```

### Step 6: Smoke Tests After Rollback

```bash
# Health check
curl -s -o /dev/null -w "%{http_code}" https://api.clario360.com/healthz
# Expected: 200

# Full health
curl -s https://api.clario360.com/api/v1/health | jq .
# Expected: all services healthy

# Login flow
curl -s -X POST https://api.clario360.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke-test@clario360.com","password":"<SMOKE_TEST_PASSWORD>"}' \
  | jq '.access_token | length > 0'
# Expected: true
```

Per-service health verification:

```bash
for svc in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo -n "$svc: "
  kubectl exec -n clario360 deploy/$svc -- wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "FAILED"
done
```

### Step 7: Post-Rollback Investigation

1. Collect logs from the failed deployment:

```bash
# Get logs from the failed pods (they may have been terminated)
kubectl logs -n clario360 -l app.kubernetes.io/part-of=clario360 \
  --previous --tail=200 \
  > /tmp/failed-deployment-logs-$(date +%Y%m%d-%H%M%S).txt 2>&1
```

2. Collect events:

```bash
kubectl get events -n clario360 \
  --sort-by='.lastTimestamp' \
  --field-selector type=Warning \
  | tail -30 \
  > /tmp/failed-deployment-events-$(date +%Y%m%d-%H%M%S).txt
```

3. Create a post-incident report:

```bash
gh issue create \
  --repo clario360/platform \
  --title "Post-Incident: Rollback from v1.5.0 to v1.4.0 $(date +%Y-%m-%d)" \
  --body "## Incident Summary
- **Time of rollback:** $(date -u +%H:%M:%S) UTC
- **Version rolled back from:** v1.5.0
- **Version rolled back to:** v1.4.0
- **Reason:** <description>
- **Impact:** <user impact description>
- **Duration of impact:** <duration>

## Root Cause
TBD - under investigation

## Action Items
- [ ] Investigate root cause
- [ ] Add regression test
- [ ] Update deployment checklist" \
  --label "incident,rollback"
```

4. Notify the team:

```bash
curl -X POST https://hooks.slack.com/services/TXXXXX/BXXXXX/XXXXXXX \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"ROLLBACK COMPLETE: Rolled back from v1.5.0 to v1.4.0 in production. Services are stable. Post-incident investigation underway.\"}"
```

---

## Verification

| Check                          | Command                                                           | Expected                          |
|-------------------------------|-------------------------------------------------------------------|-----------------------------------|
| All pods running              | `kubectl get pods -n clario360 --field-selector=status.phase!=Running` | No resources found                |
| Previous image tags restored  | `kubectl get deploy -n clario360 -o jsonpath='{range .items[*]}{.metadata.name}: {.spec.template.spec.containers[0].image}{"\n"}{end}'` | All show previous tag |
| ArgoCD in sync                | `argocd app get clario360 -o json \| jq '.status.sync.status'`   | `"Synced"`                        |
| Health endpoint               | `curl -s https://api.clario360.com/healthz`                       | 200 OK                            |
| Error rate normalized         | Prometheus `rate(http_requests_total{code=~"5.."}[5m])`           | < 0.01                           |
| Login working                 | POST `/api/v1/auth/login`                                         | Returns access_token              |

---

## Rollback of the Rollback

If the rollback itself causes issues (unlikely but possible in complex migration scenarios), escalate immediately:

1. Page the on-call engineering lead
2. Consider restoring the database from the pre-deployment backup
3. Refer to disaster recovery procedures

---

## Related Links

- [DP-001: Deploy a New Version](./DP-001-new-release.md)
- [DP-003: Emergency Hotfix Procedure](./DP-003-hotfix.md)
- [DP-004: Run Database Migrations](./DP-004-database-migration.md)
- ArgoCD Dashboard: `https://argocd.clario360.internal`
- Grafana Dashboard: `https://grafana.clario360.internal/d/clario360-overview`
- Incident Management: `https://incidents.clario360.internal`
