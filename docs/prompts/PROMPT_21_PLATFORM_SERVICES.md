# PROMPT 21 — Platform Services: Audit, Files, Notifications, Workflows & IAM

## Objective

Complete the platform-level service frontends. These are cross-cutting services used by all suites. Currently each has minimal implementation — this prompt fills in all missing CRUD operations, detail views, management UIs, and admin capabilities.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui
- **Existing types**: `types/models.ts` — has `AuditLog`, `FileItem`, `Notification`, `HumanTask`, `WorkflowInstance`, etc.
- **API helpers**: `apiGet`, `apiPost`, `apiPut`, `apiPatch`, `apiDelete`, `apiUpload` from `lib/api.ts`
- **Permissions**: Admin pages use `admin:read`/`admin:write`; workflow/notification pages use their specific permissions

---

# SECTION 1 — Audit Service Enhancements

## Constants to Add

```typescript
AUDIT_LOGS_STATS: '/api/v1/audit/logs/stats',
AUDIT_LOGS_EXPORT: '/api/v1/audit/logs/export',
AUDIT_VERIFY: '/api/v1/audit/verify',
AUDIT_PARTITIONS: '/api/v1/audit/partitions',
AUDIT_PARTITIONS_CREATE: '/api/v1/audit/partitions/create',
```

## A1. Audit Logs Page Enhancements (`admin/audit/page.tsx`)

Enhance the existing audit log viewer:

### Stats Bar
- Fetch `GET /api/v1/audit/logs/stats` → display:
  - Total log entries
  - Entries by service (bar chart or badge counts)
  - Entries by action type
  - Entries today vs yesterday (delta)

### Detail View
- Click any log row → slide-out detail panel
- Fetch `GET /api/v1/audit/logs/{id}` → show:
  - All fields: user, action, resource type/id, service, severity badge
  - IP address, user agent
  - Correlation ID (linkable to related logs)
  - Old Value vs New Value (side-by-side JSON diff if both present)
  - Entry hash and previous hash (chain integrity)
  - Metadata (formatted JSON)

### Resource Timeline
- Button per log entry or resource: "View Timeline"
- Fetch `GET /api/v1/audit/logs/timeline/{resourceId}` → timeline component showing all changes to that resource chronologically

### Export
- **Export** button in toolbar
- `GET /api/v1/audit/logs/export?format=csv|json&...filters`
- Apply current filters to export
- Download file via blob URL

### Chain Verification (Admin)
- **Verify Integrity** button (admin only)
- `POST /api/v1/audit/verify` → show results:
  - Chain valid: success message with badge
  - Chain broken: error with details of where the break occurred

### Partition Management (Admin)
- Separate admin section or tab
- `GET /api/v1/audit/partitions` → DataTable of partitions (name, date range, record count, size)
- **Create Partition** → `POST /api/v1/audit/partitions/create` — Admin dialog

### Enhanced Filters
Add to existing filters:
- Service filter (multi-select)
- Severity filter (info, warning, high, critical)
- Action type filter
- Resource type filter
- Date range picker
- Correlation ID search
- User filter

**Components:**
```
components/admin/audit/
  audit-stats-bar.tsx
  audit-log-detail-panel.tsx       — Side panel with full details
  audit-log-diff-viewer.tsx        — Old vs new value diff
  audit-resource-timeline.tsx      — Timeline for resource ID
  audit-export-button.tsx          — Export with format selection
  audit-verify-button.tsx          — Chain integrity verification
  audit-partition-table.tsx        — Partition management (admin)
  audit-partition-create-dialog.tsx
```

---

# SECTION 2 — File Service (Full Management)

## Constants to Add

```typescript
FILES_UPLOAD_PRESIGNED: '/api/v1/files/upload/presigned',
FILES_UPLOAD_CONFIRM: '/api/v1/files/upload/confirm',
FILES_QUARANTINE: '/api/v1/files/quarantine',
FILES_STATS: '/api/v1/files/stats',
```

## B1. Files Page Enhancements (`files/page.tsx`)

Currently shows basic file list. Enhance with full file management:

### File List Enhancements
- DataTable columns: name, content_type, size (formatted), virus_scan_status badge, encrypted badge, suite, version, uploaded_by, created_at
- Filters: suite, content_type, virus_scan_status, encrypted, date range
- **Upload File** button (already exists — verify progress indicator works)
- Per-row actions: Download, View Details, View Versions, Delete

### File Detail (slide-out panel)
- Fetch `GET /api/v1/files/{id}` → show all FileUploadRecord fields:
  - Original name, sanitized name
  - Content type, size (human readable)
  - Checksum SHA-256 (copyable)
  - Encrypted status, virus scan status badge
  - Suite, entity type, entity ID (linkable)
  - Tags (editable)
  - Version number
  - Lifecycle policy, expires_at
  - Upload metadata

### File Download
- **Download** button → `GET /api/v1/files/{id}/download`
- **Presigned Download** → `GET /api/v1/files/{id}/presigned` — generates temporary URL

### File Versions
- `GET /api/v1/files/{id}/versions` → version history table
- Download any previous version

### Access Log
- `GET /api/v1/files/{id}/access-log` → table of who accessed this file, when, action type

### Delete File
- `DELETE /api/v1/files/{id}` → confirm dialog with warning about permanent deletion

### Presigned Upload Flow
- Alternative to direct upload for large files:
  1. `POST /api/v1/files/upload/presigned` → get presigned URL
  2. Upload directly to storage using presigned URL (client-side)
  3. `POST /api/v1/files/upload/confirm` → confirm upload

### Admin Features

#### Quarantine Management
- **Quarantine** tab (admin only)
- `GET /api/v1/files/quarantine` → DataTable of quarantined files
  - Columns: name, content_type, size, quarantine_reason, uploaded_by, date
- Per-row actions:
  - **Resolve** → `POST /api/v1/files/quarantine/{id}/resolve` — Dialog with action (release, delete, keep quarantined) + notes
  - **Rescan** → `POST /api/v1/files/{id}/rescan` — Re-run virus scan

#### File Statistics
- `GET /api/v1/files/stats` → display:
  - Total files, total size
  - By content type (pie chart)
  - By suite (bar chart)
  - Quarantined count
  - Scan statistics

**Components:**
```
components/files/
  file-columns.tsx
  file-detail-panel.tsx
  file-download-button.tsx
  file-version-table.tsx
  file-access-log-table.tsx
  file-delete-dialog.tsx
  file-upload-presigned.tsx        — Presigned upload flow
  file-quarantine-table.tsx        — Admin quarantine management
  file-quarantine-resolve-dialog.tsx
  file-rescan-button.tsx
  file-stats-dashboard.tsx
```

---

# SECTION 3 — Notification Service Enhancements

## Constants to Add

```typescript
NOTIFICATIONS_WEBHOOKS: '/api/v1/notifications/webhooks',
NOTIFICATIONS_TEST: '/api/v1/notifications/test',
NOTIFICATIONS_DELIVERY_STATS: '/api/v1/notifications/delivery-stats',
NOTIFICATIONS_RETRY_FAILED: '/api/v1/notifications/retry-failed',
```

## C1. Notification Center Enhancements (`notifications/page.tsx`)

### Mark Read/Delete
Wire up existing UI to actual endpoints:
- **Mark as Read** → `PUT /api/v1/notifications/{id}/read`
- **Mark All as Read** → `PUT /api/v1/notifications/read-all`
- **Delete** → `DELETE /api/v1/notifications/{id}`

### Notification Preferences
Wire up the preferences form:
- **Load Preferences** → `GET /api/v1/notifications/preferences`
- **Save Preferences** → `PUT /api/v1/notifications/preferences`

## C2. Notification Settings Page Enhancements (`settings/notifications/page.tsx`)

### Webhook Management
New section on notification settings page (or separate admin page):

- **Webhook List** → `GET /api/v1/notifications/webhooks` — DataTable:
  - Columns: name, URL (truncated), events subscribed, status (active/inactive), last delivered, created_at
- **Create Webhook** → `POST /api/v1/notifications/webhooks` — Dialog:
  - Name, URL (required), Secret (for HMAC, auto-generated option), Events (multi-select), Active toggle
- **Edit Webhook** → `PUT /api/v1/notifications/webhooks/{id}`
- **Delete Webhook** → `DELETE /api/v1/notifications/webhooks/{id}`
- **Test Webhook** → Send test ping to verify URL

### Test Notification
- **Send Test** button → `POST /api/v1/notifications/test` — Dialog:
  - Channel (select: in_app, email, webhook)
  - Priority (select)
  - Message (text)
  - Verifies delivery pipeline

### Delivery Statistics (Admin)
- `GET /api/v1/notifications/delivery-stats` → dashboard:
  - Total sent, delivered, failed
  - By channel (in_app, email, webhook, push)
  - Delivery success rate
  - Recent failures list
- **Retry Failed** button → `POST /api/v1/notifications/retry-failed` — Retries all failed deliveries

**Components:**
```
components/notifications/
  notification-mark-read-button.tsx
  notification-delete-button.tsx
  notification-mark-all-read.tsx

  webhooks/
    webhook-columns.tsx
    webhook-create-dialog.tsx
    webhook-edit-form.tsx
    webhook-delete-dialog.tsx
    webhook-test-button.tsx

  admin/
    notification-test-dialog.tsx
    delivery-stats-dashboard.tsx
    retry-failed-button.tsx
```

---

# SECTION 4 — Workflow Engine Enhancements

## Constants to Add

```typescript
WORKFLOWS_DEFINITIONS: '/api/v1/workflows/definitions',
WORKFLOWS_TEMPLATES: '/api/v1/workflows/templates',
```

## D1. Workflow Instances Page Enhancements (`workflows/page.tsx`)

### Instance Actions
Add per-instance actions:
- **Cancel** → Currently has dialog, verify it calls `DELETE /api/v1/workflows/instances/{id}` or `POST .../cancel`
- **View Detail** → Verify link to `workflows/[id]`

### Create Instance
- **Start Workflow** button → `POST /api/v1/workflows/instances` — Dialog:
  - Definition (select from definitions list)
  - Variables (dynamic form based on definition input schema)

## D2. Workflow Definitions Page (`workflows/definitions/page.tsx`) — NEW PAGE

List and manage workflow definitions (admin feature):

- DataTable → `GET /api/v1/workflows/definitions`
  - Columns: name, description, version, step count, active instances, created_at
- **Create Definition** → `POST /api/v1/workflows/definitions` — Complex form:
  - Name, Description
  - Steps (dynamic array): step_id, name, type (human_task, service_task, condition, parallel_gateway, timer, end), config
  - Visual step flow preview
- **Edit** → `PUT /api/v1/workflows/definitions/{id}`
- **Delete** → `DELETE /api/v1/workflows/definitions/{id}`

## D3. Workflow Templates Page (`workflows/templates/page.tsx`) — NEW PAGE

- DataTable → `GET /api/v1/workflows/templates`
- Pre-built workflow templates for common processes
- **Use Template** → Creates a new definition from template

## D4. Task Detail Page Enhancements (`workflows/tasks/[id]/page.tsx`)

Verify all task actions work:
- **Claim** → Assign to current user
- **Complete** → Submit form data
- **Reject** → With reason
- **Delegate** → To another user
- **Assign** → `POST /api/v1/workflows/tasks/{id}/assign` — Specific user assignment

**Components:**
```
components/workflows/
  workflow-start-dialog.tsx        — Create new instance
  workflow-definition-columns.tsx
  workflow-definition-form.tsx     — Step builder
  workflow-step-flow-preview.tsx   — Visual step diagram
  workflow-template-columns.tsx
  workflow-template-use-dialog.tsx
```

---

# SECTION 5 — IAM Service Enhancements

## Constants to Add

```typescript
USERS: '/api/v1/users',
TENANTS: '/api/v1/tenants',
API_KEYS: '/api/v1/api-keys',
ADMIN_TENANTS_PROVISION: '/api/v1/admin/tenants/provision',
ADMIN_TENANTS_DEPROVISION: '/api/v1/admin/tenants', // /{id}/deprovision
```

## E1. User Management Enhancements (`admin/users/page.tsx`)

Verify existing CRUD works, add:
- **Invitation Management** tab or section:
  - List invitations → `GET /api/v1/invitations`
  - Send invitation → `POST /api/v1/invitations` — Dialog with email, role
  - Delete invitation → `DELETE /api/v1/invitations/{id}`

## E2. Tenant Management Page (`admin/tenants/page.tsx`) — NEW PAGE

Admin-only page for super-admin tenant management:

- DataTable → `GET /api/v1/tenants` — List all tenants
  - Columns: name, slug, status, max_users, created_at
- **Create Tenant** → Dialog with name, slug, settings
- **Edit Tenant** → `PUT /api/v1/tenants/{id}`
- **Provision Tenant** → `POST /api/v1/admin/tenants/provision` — Creates database, runs migrations
- **Deprovision Tenant** → `POST /api/v1/admin/tenants/{id}/deprovision` — DANGER: confirm dialog with type-to-confirm

## E3. API Key Management Page (`admin/api-keys/page.tsx` or `settings/api-keys/page.tsx`) — NEW PAGE

- DataTable → `GET /api/v1/api-keys`
  - Columns: name, prefix (masked key), permissions, last_used, expires_at, created_at
- **Create API Key** → `POST /api/v1/api-keys` — Dialog:
  - Name (required)
  - Permissions (multi-select or permission tree)
  - Expiry (date picker or "never")
  - Show generated key ONCE after creation (copy button, warning it won't be shown again)
- **Revoke** → `DELETE /api/v1/api-keys/{id}` — Confirm dialog

**Components:**
```
components/admin/
  tenants/
    tenant-columns.tsx
    tenant-create-dialog.tsx
    tenant-edit-form.tsx
    tenant-provision-dialog.tsx
    tenant-deprovision-dialog.tsx  — Type-to-confirm danger dialog

  api-keys/
    api-key-columns.tsx
    api-key-create-dialog.tsx     — Shows key once, copy button
    api-key-revoke-dialog.tsx
    api-key-permissions-select.tsx

  invitations/
    invitation-columns.tsx
    invitation-send-dialog.tsx
    invitation-delete-dialog.tsx
```

---

## Sidebar Navigation Updates

```
Admin
├── Users              (existing — add invitations)
├── Roles              (existing)
├── Tenants            (NEW — super-admin only)
├── API Keys           (NEW)
├── Audit Logs         (existing — enhance)
└── AI Governance      (existing — enhance in Prompt 22)

Workflows
├── Instances          (existing — enhance)
├── Tasks              (existing)
├── Definitions        (NEW — admin)
└── Templates          (NEW — admin)
```

Add new pages:
```
app/(dashboard)/admin/tenants/page.tsx          — NEW
app/(dashboard)/admin/api-keys/page.tsx         — NEW (or settings/api-keys)
app/(dashboard)/workflows/definitions/page.tsx  — NEW
app/(dashboard)/workflows/templates/page.tsx    — NEW
```

---

## Testing Requirements

### Audit
- Test detail panel renders all fields including JSON diff
- Test resource timeline renders chronologically
- Test export generates downloadable file
- Test chain verification shows success/failure
- Test partition management CRUD

### Files
- Test file download triggers browser download
- Test version history shows all versions
- Test access log renders correctly
- Test quarantine resolve workflow
- Test presigned upload flow
- Test file stats dashboard

### Notifications
- Test mark read/delete API calls
- Test preferences save/load cycle
- Test webhook CRUD
- Test delivery stats rendering
- Test retry failed action

### Workflows
- Test workflow start with variable form
- Test definition CRUD (if implementing full designer, keep it simple)
- Test task claim/complete/reject/delegate

### IAM
- Test invitation send/list/delete
- Test API key creation shows key once
- Test tenant management with proper permission gates
- Test deprovision type-to-confirm dialog

## Validation Checklist

- [ ] `npm run build` passes with zero errors
- [ ] All audit endpoints have frontend calls
- [ ] All file service endpoints have frontend calls
- [ ] All notification endpoints have frontend calls
- [ ] All workflow endpoints have frontend calls
- [ ] All IAM management endpoints have frontend calls
- [ ] Admin pages are gated behind admin permissions
- [ ] API key is shown only once after creation
- [ ] Tenant deprovision has type-to-confirm safety
- [ ] File quarantine management is admin-only
- [ ] Chain verification result is clearly displayed
