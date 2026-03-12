# Frontend / Backend Feature Parity Report

_Last reviewed: 12 March 2026_

## Summary

This report compares the user-facing feature surface implemented in the backend with what is currently exposed through the frontend application.

## Overall Verdict

- The frontend is **not yet at full parity** with the backend.
- The previous draft undercounted the platform services; this revision now includes **IAM/Auth**, **Audit**, **Workflow**, **Notification**, and **File Service** alongside the main suites.
- **Cyber**, **Data**, and **Acta** have the strongest end-user workflow coverage among the domain suites.
- **Visus**, **Audit**, **File Service**, and parts of **AI Governance administration** still have substantial backend-ready capabilities that are not fully surfaced in the UI.
- **Lex** has improved materially on the contract lifecycle path, but document administration and compliance-management tooling still trail the backend.
- Some items that initially looked missing are already wired in the frontend, especially **audit chain verification**, **notification read/delete actions**, and several **workflow task actions**.

## Method

This comparison is based on:

- backend route surfaces in:
	- `backend/internal/acta/handler/routes.go`
	- `backend/internal/lex/handler/routes.go`
	- `backend/internal/cyber/handler/routes.go`
	- `backend/internal/data/handler/routes.go`
	- `backend/internal/visus/handler/routes.go`
	- `backend/internal/aigovernance/handler/routes.go`
- frontend dashboard pages and API wiring under:
	- `frontend/src/app/(dashboard)/`
	- `frontend/src/lib/constants.ts`
	- `frontend/src/lib/enterprise/`
	- `frontend/src/lib/suite-api.ts`

This report evaluates **user-facing parity**, not internal-only backend behavior such as schedulers, consumers, background workers, seeders, or internal governance helpers.

## Reconciliation Notes

The earlier report missed several already-implemented frontend flows. Verified examples include:

- **Audit**: hash-chain verification is wired from the audit page via `POST /api/v1/audit/verify`.
- **Notifications**: save preferences, mark-as-read, mark-all-read, and delete are wired.
- **Workflow Engine**: claim, complete, reject, delegate, cancel, retry, suspend, and resume flows are wired in workflow/task UI.
- **Acta**: attendance, agenda, voting, minutes generation/update/submit/approve/publish, attachments, and action item actions are wired; it is not just list/create coverage.
- **AI Governance**: there is more than a single overview page; model detail, shadow comparison, drift, performance, prediction logs, explanations, promote, rollback, and feedback are all surfaced.

## Parity Matrix

| Suite | Backend Coverage | Frontend Coverage | Parity Status | Notes |
|---|---:|---:|---:|---|
| IAM / Auth | High | High | Partial / Strong | Core auth, user, role, onboarding, MFA, and settings flows are surfaced; tenant admin, API keys, and advanced admin flows are not. |
| Audit | Medium | Low-Medium | Partial / Weak | Log listing and hash verification exist, but broader audit analytics/export/admin tooling are missing. |
| Workflow Engine | Medium | Medium | Partial | Instance and task operations are better covered than first assumed, but definition/template management is still absent. |
| Notification | Medium | Medium | Partial | Listing, unread count, preferences save, mark-read, mark-all-read, and delete exist; webhook/admin observability gaps remain. |
| File Service | Medium | Low | Partial / Weak | Listing and upload are present, but most lifecycle, download, metadata, and admin functions are missing. |
| Acta | High | High | Partial / Strong | Most major meeting-management flows are surfaced; some lifecycle and CRUD edges are still missing. |
| Lex | High | Medium-High | Partial | Core contract lifecycle, analysis, and review flows are now wired; document and compliance administration still lag. |
| Cyber | Very High | High | Partial / Strong | Many operational views exist, but not every backend action is surfaced. |
| Data | Very High | High | Partial / Strong | Large and broad frontend exists, but backend surface is larger still. |
| Visus | High | Medium | Partial / Weak | Read/report views exist, but authoring and executive endpoints are underrepresented. |
| AI Governance | High | Medium-High | Partial | Good monitoring and review coverage, incomplete administration coverage. |

## Suite-by-Suite Findings

### IAM / Auth

**Frontend surfaced**

- Login, logout, refresh, forgot/reset password, MFA setup/verify/disable
- Current-user profile update and password/session settings
- User management and role management pages under `frontend/src/app/(dashboard)/admin/users` and `frontend/src/app/(dashboard)/admin/roles`
- Onboarding registration, verification, wizard steps, and invitation accept/validate flows

**Not clearly surfaced in frontend**

- Tenant administration CRUD
- API key management lifecycle
- Invitation management outside onboarding-style flows
- Super-admin tenant provisioning / deprovisioning
- OAuth / PKCE login provider UX

**Assessment**

- IAM/Auth is **mostly covered for tenant end users and org admins**, but **platform-admin features are not at parity**.

### Audit

**Frontend surfaced**

- Audit log list with filters
- Detail panel from row selection
- Hash-chain verification action from the audit page

**Not clearly surfaced in frontend**

- Audit stats dashboard
- Dedicated log detail route / resource timeline view
- Export flows
- Partition management and other audit-admin operations

**Assessment**

- Audit is **better than a simple list-only view**, but still has **major analytics/admin gaps**.

### Workflow Engine

**Frontend surfaced**

- Workflow instance list and detail pages
- Task queue, task counts, claim flow, complete flow, reject flow, delegate flow
- Instance actions including cancel, retry, suspend, and resume

**Not clearly surfaced in frontend**

- Workflow definition CRUD / designer UI
- Workflow template management
- Manual workflow-instance creation / editing / deletion
- Broader assignment/admin orchestration features

**Assessment**

- Workflow frontend coverage is **meaningfully stronger than initially reported**, but it is still **not full parity** because design-time/admin capabilities are missing.

### Notification

**Frontend surfaced**

- Notification list, unread count, real-time updates
- Notification preferences page with save
- Mark single notification read
- Mark all read
- Delete notification

**Not clearly surfaced in frontend**

- Webhook management
- Test notification tooling
- Delivery statistics dashboards
- Retry-failed-deliveries operations

**Assessment**

- Notification coverage is **moderate**, not minimal; the main gaps are **admin and delivery-operations tooling**.

### File Service

**Frontend surfaced**

- File listing page
- Upload flows used from suite pages such as Acta attachments and action evidence

**Not clearly surfaced in frontend**

- File metadata detail page
- Download / presigned download workflows
- Delete file lifecycle
- File version history
- Access log
- Quarantine management and resolve actions
- File stats and rescan admin flows

**Assessment**

- File service remains **one of the larger platform-service gaps**.

### Acta

**Backend capabilities present**

- Committees: create, list, get, update, delete, add/update/remove members
- Meetings: create, list, get, update, delete, start, end, postpone, attendance, bulk attendance, attachments
- Agenda: create, list, reorder, update, delete, notes, vote
- Minutes: create, get latest, list versions, generate, update, submit, request revision, approve, publish
- Action items: overdue, mine, stats, create, list, get, update, status update, extend
- Compliance: run, results, report, score
- Dashboard: get

**Frontend surfaced**

- Dashboard, meetings list/detail/calendar, committees list/detail, action items, compliance pages under `frontend/src/app/(dashboard)/acta`
- Meeting detail supports attendance, agenda, voting, AI minutes generation, update, submit, approve, publish, attachments, and extracted action item creation

**Not clearly surfaced in frontend**

- Committee update and delete flows
- Meeting update and delete flows
- Manual minutes create flow distinct from AI generation/update
- Minutes revision-request flow
- Some committee/meeting administrative actions appear backend-only

**Assessment**

- Acta is one of the closest suites to parity, but it is still **not 100% complete**.

### Lex

**Backend capabilities present**

- Contracts: expiring, stats, search, create, list, get, update, delete
- Contract analysis: analysis, upload document, analyze, versions, renew, start review
- Clauses: list, get, risk summary, review
- Documents: create, list, get, update, delete, upload version, list versions
- Compliance: rules CRUD, run, alerts get/list/update, dashboard, score
- Workflows listing
- Dashboard

**Frontend surfaced**

- Overview page
- Contracts list and detail page
- Contract create, edit, and delete flows
- Contract analysis trigger, compliance run trigger, status change, renewal, and review-workflow start
- Contract version history UI and upload-new-version flow
- Clause review action from contract detail
- Documents list page
- Compliance page and dashboard widgets
- Search support via command palette

**Not clearly surfaced in frontend**

- Document create / edit / delete / version upload flows
- Compliance rule create / update / delete flows
- Alert status update flow
- Workflow listing UI tied to Lex backend

**Assessment**

- Lex remains **partial**, but the highest-value contract lifecycle path is now substantially closer to parity; the remaining gaps are concentrated in document administration, compliance rule management, alert operations, and workflow-listing views.

### Cyber

**Backend capabilities present**

- Assets: stats, count, scan, scan listing/detail/cancel, bulk create/update/delete, CRUD, tags, relationships, vulnerabilities
- Alerts: stats, count, comments, timeline, related, get, list, status, assign, escalate, comment, merge
- Rules: templates, CRUD, toggle, test, feedback
- Threats and indicators: stats, threat detail/list, status update, indicator list/add/check/bulk import
- MITRE: tactics, techniques, coverage
- Vulnerabilities: stats, aging, top CVEs, get, list, status
- Risk: score, trend, recalculate, heatmap, top risks, recommendations
- Dashboard: dashboard, KPIs, timelines, MTTR, workload, attacked assets, heatmap, trends
- Remediation: CRUD and full lifecycle including submit/approve/reject/revision/dry-run/execute/verify/rollback/close/audit trail
- DSPM: data assets, scans, classification, exposure, dependencies, dashboard
- vCISO: briefing, history, recommendations, report, posture summary
- CTEM: assessments, dashboard, exposure score/history/calc, findings, remediation groups, reports, compare

**Frontend surfaced**

- Pages for dashboard, assets, asset scans, alerts, rules, CTEM, DSPM, remediation, risk heatmap, vCISO, threats, MITRE
- Many dialogs and detail views for scans, alerts, rules, remediation lifecycle, CTEM assessments, DSPM, and vCISO

**Not clearly surfaced in frontend**

- Alert merge flow
- Related-alerts UI
- Threat status update and indicator management breadth
- Some vulnerability endpoints such as top CVEs / status workflows
- Risk recalculate, top risks, and recommendations endpoints
- Some CTEM actions such as compare assessments and remediation-group execution may not be fully surfaced
- Some vCISO history/recommendation/posture-specific endpoints are only partially represented

**Assessment**

- Cyber has **strong frontend coverage**, but the backend still exposes capabilities beyond the current UI.

### Data

**Backend capabilities present**

- Sources: source types, create/list/get/update/delete, status, test, discover, schema, sync, sync history, stats
- Models: create/list/derive/get/update/delete/validate/versions/lineage
- Pipelines: stats, active, CRUD, run, pause, resume, runs, run detail, logs
- Quality: score, trend, dashboard, rules CRUD, run, results
- Contradictions: scan, scan history, stats, dashboard, list/get/status/resolve
- Lineage: full graph, entity graph, upstream, downstream, impact, record, delete edge, search, stats
- Dark data: scan, scan history, stats, dashboard, list/get/status/govern
- Analytics: query, explore, explain, saved queries CRUD, run saved, audit
- Dashboard

**Frontend surfaced**

- Pages for dashboard, sources, models, pipelines, quality, contradictions, lineage, dark data, analytics
- Wizards and detail views for many source/pipeline flows
- Dedicated visuals for lineage, quality, contradictions, and data operations

**Not clearly surfaced in frontend**

- Some specialized lineage mutation/admin actions such as record/delete edge
- Some analytics endpoints like explain/audit may not be fully surfaced in page-level UX
- Some source/model lifecycle operations may exist only as API helpers or secondary flows

**Assessment**

- Data appears **broadly implemented in the frontend**, but due to the size of the backend API surface, it should still be treated as **high coverage, not proven full parity**.

### Visus

**Backend capabilities present**

- Dashboards: create, list, get, update, delete, duplicate, share
- Widgets: create, list, update, delete, data, layout update, widget types
- KPIs: create, list, summary, snapshot trigger, get, update, delete, history
- Alerts: list, count, stats, get, status update
- Reports: create, list, get, update, delete, generate, snapshots, latest snapshot, snapshot detail
- Executive: view, summary, health

**Frontend surfaced**

- Overview page
- KPI page
- Alerts page
- Reports page

**Not clearly surfaced in frontend**

- Dashboard authoring CRUD
- Dashboard duplicate/share flows
- Widget create/update/delete/layout management
- Widget data explorer / widget type management
- KPI create/update/delete/history/snapshot flows
- Report create/update/delete and snapshot browsing
- Executive view / summary / health pages

**Assessment**

- Visus frontend is **meaningfully behind** the backend feature surface.

### AI Governance

**Backend capabilities present**

- Models: register, list, get, update
- Versions: create, list, get
- Lifecycle: promote, retire, rollback, lifecycle history
- Shadow: start, stop, latest comparison, history, divergences
- Drift: latest, history, performance
- Predictions and explanations
- Dashboard

**Frontend surfaced**

- AI governance overview page
- Model detail page with lifecycle history, shadow comparison, divergences, drift, performance, prediction logs, explanations
- Promote and rollback dialogs
- Feedback submission

**Not clearly surfaced in frontend**

- Register model flow
- Update model flow
- Create version flow
- Get version detail as a dedicated management flow
- Retire version flow
- Stop shadow flow
- Some administrative registry operations beyond monitoring and promotion

**Assessment**

- AI governance has **solid observability and review UX**, but **not full administrative parity**.

## Notes Against the Earlier Gap List

The following items from the supplemental gap list are already implemented in the frontend and should not be counted as missing:

- Audit hash-chain verification
- Notification preference save
- Notification mark-read / mark-all-read / delete
- Workflow task claim / complete / reject / delegate
- Workflow instance cancel / retry / suspend / resume
- Large portions of Acta meeting operations including attendance, attachments, agenda management, minutes lifecycle, compliance pages, and dashboard wiring
- AI Governance shadow comparison, drift monitoring, prediction log viewer, feedback, promote, and rollback

The following themes from the supplemental gap list are directionally correct and are now reflected in this report:

- Lex remains substantially under-surfaced
- Visus authoring/admin flows are still largely absent
- File service admin/detail flows are missing
- Audit analytics/export/admin tooling remains thin
- AI governance lacks full registry/version administration UI

## Biggest Missing Areas

### Highest frontend gaps

1. **Lex**
2. **Visus**
3. **File Service**
4. **Audit analytics / admin tooling**
5. **AI Governance administration**

### Best-covered suites

1. **Cyber**
2. **Acta**
3. **Data**
4. **Workflow task/instance operations**
5. **IAM/Auth core tenant flows**

## Final Verdict

- The frontend is **substantial and mature**, but it does **not implement every single backend feature or capability**.
- If the goal is true suite-by-suite parity, the highest-value backlog should focus on:
	- **Lex analysis and workflow operations**
	- **Visus authoring and executive endpoints**
	- **File service lifecycle and admin tooling**
	- **Audit analytics/export/admin tooling**
	- **AI governance admin flows**
	- **Remaining Acta and Cyber lifecycle edge actions**

## Suggested Next Step

The next useful artifact would be a **gap tracker** with columns like:

- backend capability
- frontend status
- priority
- recommended page / component
- API already available

That would turn this parity audit into an implementation backlog.
