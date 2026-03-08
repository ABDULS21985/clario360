# PROMPT: Frontend End-to-End Integration Testing & Bug Fixing

You are testing the Clario360 Next.js 14 frontend at `/Users/mac/clario360/frontend`.

## CRITICAL RULES (VIOLATIONS WILL BREAK THE BUILD)

- Test framework: Vitest + @testing-library/react + MSW 2.x
- Test setup file: `src/__tests__/setup.ts` (polyfills ResizeObserver, PointerEvent, IntersectionObserver)
- MSW 2.x syntax: `import { http, HttpResponse } from 'msw'` — NOT `rest` (v1 syntax)
- Auth tokens: access tokens in memory (`lib/auth.ts`), refresh tokens in httpOnly cookies
- API calls: `apiGet/apiPost/apiPut/apiPatch/apiDelete` from `lib/api.ts` (Axios-based)
- Stores: Zustand — test by rendering components, NOT by calling store actions directly
- Brand colors: primary #1B5E20, gold #C6A962, teal #0D4B4F
- FormField: puts `id` on `<div>` not `<input>` — use `getByPlaceholderText` in tests, NOT `getByLabelText`
- Combobox: uses `onChange` prop (NOT `onValueChange`), only `{label, value}` options
- DataTable: `onSortChange` is required even if unused — pass `() => undefined`
- LoadingSkeleton variants: 'card'|'table-row'|'list-item'|'text'|'avatar'|'chart' (no 'detail')
- tsconfig target: ES2017 (needed for Map iterators)
- useApiMutation: 409 errors go to toast, not form field errors
- Commands: `npm run build`, `npm run test`, `npm run type-check`

## PHASE 1: BUILD VERIFICATION

Run these first. Fix every error before proceeding:

```bash
cd /Users/mac/clario360/frontend
npm run type-check    # Must pass with zero TypeScript errors
npm run build         # Must build all 72 pages successfully
npm run test          # Must pass all existing tests
```

Report: total tests, build output (X/X pages), any failures.

## PHASE 2: AUTH FLOW INTEGRATION TESTS

Test the complete auth lifecycle using MSW to mock backend:

### 2.1 Login Flow (`src/__tests__/integration/login-flow.test.tsx`)

Mock endpoints:

```typescript
http.post('http://localhost:8080/api/v1/auth/login', async ({ request }) => {
  const body = await request.json();
  // Test various scenarios...
})
```

Test cases:
- **Happy path**: Enter email + password → submit → verify `auth-store` has user, access token set in memory
- **Invalid credentials**: 401 response → verify error message shown, form not cleared
- **MFA required**: Response `{ mfa_required: true, mfa_token: "..." }` → verify MFA input appears
- **MFA verification**: Enter TOTP code → submit → verify login completes
- **Rate limited**: 429 response → verify "too many attempts" message, Retry-After shown
- **Network error**: Mock network failure → verify error state
- **Redirect after login**: Visit `/cyber/alerts` while logged out → login → verify redirect to `/cyber/alerts`
- **Session expired dialog**: Trigger `clario360:session-expired` event → verify dialog appears

### 2.2 Registration Flow

- Fill form: email, password, first_name, last_name, company
- Submit → verify OTP input appears
- Enter OTP → verify redirect to setup wizard
- Verify password strength meter updates in real-time

### 2.3 Token Refresh (`src/__tests__/integration/token-refresh.test.tsx`)

- Make API call → receive 401 → verify refresh called → verify original request retried
- Multiple concurrent 401s → verify only ONE refresh call (mutex pattern in `api.ts`)
- Refresh fails → verify `SessionExpiredDialog` shown

### 2.4 Session Hydration

- Test `AuthProvider` hydration: mock `GET /api/auth/session` → verify user/tenant populated
- Test page refresh: verify access token restored from BFF session route

## PHASE 3: REAL CRUD OPERATION TESTS

For EACH domain, write integration tests that render the actual page component, mock the
API, and verify the full create→read→update→delete cycle in the UI.

### 3.1 User Management (`/admin/users`)

**File**: `src/__tests__/integration/admin/user-crud.test.tsx`

Mock these endpoints:

```
GET  /api/v1/users           → PaginatedResponse<User>
POST /api/v1/users           → User
GET  /api/v1/users/{id}      → User
PUT  /api/v1/users/{id}      → User
DELETE /api/v1/users/{id}    → 204
GET  /api/v1/roles           → PaginatedResponse<Role>
POST /api/v1/users/{id}/roles → 200
```

Test cases:
- **List**: Render page → verify table shows users with columns: name, email, status, roles, actions
- **Search**: Type in search → verify API called with `?search=` parameter
- **Pagination**: Click next page → verify API called with `?page=2`
- **Create**: Click "Add User" → fill form (email, name, password, roles) → submit → verify POST called → verify table refreshes
- **Edit**: Click edit on user → verify form pre-filled → change name → submit → verify PUT called
- **Delete**: Click delete → confirm dialog → verify DELETE called → verify user removed from table
- **Role assignment**: Open user → assign role → verify POST to `/users/{id}/roles`
- **Validation**: Submit empty form → verify field errors shown
- **Duplicate email**: Mock 409 → verify toast error

### 3.2 Role Management (`/admin/roles`)

**File**: `src/__tests__/integration/admin/role-crud.test.tsx`

Test cases:
- List roles with permission counts
- Create role: name, description, permission selection (PermissionTree with indeterminate checkboxes)
- Edit role: change permissions → verify PUT called
- Delete role: verify cannot delete system roles (button disabled/hidden)
- Wildcard permission: select parent → verify children auto-selected

### 3.3 Audit Logs (`/admin/audit`)

**File**: `src/__tests__/integration/admin/audit-logs.test.tsx`

Mock: `GET /api/v1/audit/logs`, `GET /api/v1/audit/logs/{id}`, `GET /api/v1/audit/logs/export`

Test cases:
- List with filters: action, date range, user, resource type
- Click row → detail panel shows old_value/new_value diff
- Export: click export → verify request, loading state, download

### 3.4 Cyber Assets (`/cyber/assets`)

**File**: `src/__tests__/integration/cyber/asset-crud.test.tsx`

Mock endpoints:

```
GET    /api/v1/cyber/assets          → PaginatedResponse<Asset>
POST   /api/v1/cyber/assets          → Asset
GET    /api/v1/cyber/assets/{id}     → Asset with relationships
PUT    /api/v1/cyber/assets/{id}     → Asset
DELETE /api/v1/cyber/assets/{id}     → 204
PATCH  /api/v1/cyber/assets/{id}/tags → Asset
```

Test cases:
- **List**: Table with columns: name, type, IP, criticality, status, tags, last_seen
- **Create**: Form with fields matching Asset type (name, type enum, ip_address, hostname, os, criticality, tags[])
- **Edit**: Pre-fill all fields → update hostname → verify PUT
- **Delete**: Confirm dialog → verify soft delete
- **Tag management**: Add/remove tags via PATCH endpoint
- **Bulk operations**: Select multiple → bulk delete → verify all removed
- **Type filter**: Filter by type=server → verify API query param
- **Severity filter**: Filter by criticality=critical → verify

### 3.5 Cyber Alerts (`/cyber/alerts`)

**File**: `src/__tests__/integration/cyber/alert-crud.test.tsx`

Test cases:
- List with severity badges (critical=red, high=orange, medium=yellow, low=blue)
- Acknowledge alert → status changes to "acknowledged"
- Resolve alert → status changes to "resolved"
- Filter by severity, status, date range
- Click alert → detail page with full description, related assets, timeline

### 3.6 CTEM Assessments (`/cyber/ctem`)

**File**: `src/__tests__/integration/cyber/ctem-crud.test.tsx`

Test cases:
- List assessments with status badges
- Create assessment: name, scope (asset selection), schedule
- Assessment detail: findings list, severity breakdown, validation actions
- Export assessment report

### 3.7 Vulnerabilities (`/cyber/assets` detail page)

Test vulnerability CRUD within asset context:
- List vulnerabilities for asset
- Create vulnerability: title, CVE ID, severity, description
- Status transitions: open → in_progress → resolved
- Verify CVE auto-enrichment data shown

### 3.8 Data Sources (`/data/sources`)

**File**: `src/__tests__/integration/data/source-crud.test.tsx`

Test cases:
- List data sources with connection status indicators
- Create source: select type (PostgreSQL, ClickHouse, etc.), enter connection details
- Test connection button → verify success/failure feedback
- Edit connection details
- Delete source with confirmation

### 3.9 Data Pipelines (`/data/pipelines`)

**File**: `src/__tests__/integration/data/pipeline-crud.test.tsx`

Test cases:
- List pipelines with status (enabled/disabled), last run status, schedule
- Create pipeline: add transforms (rename, cast, filter, aggregate), configure each
- Execute pipeline → verify run created with status "running"
- Pipeline detail: run history, logs, stage visualization

### 3.10 Workflows (`/workflows`)

**File**: `src/__tests__/integration/workflows/workflow-crud.test.tsx`

Test cases:
- List workflow instances with status, progress (X/Y steps complete)
- Instance detail: step timeline with completed/running/pending states
- Cancel workflow → confirm → verify status change

### 3.11 Human Tasks (`/workflows/tasks`)

**File**: `src/__tests__/integration/workflows/task-crud.test.tsx`

Test cases:
- List tasks with tabs: pending, claimed, completed, overdue, escalated
- Claim task → verify status changes to "claimed"
- Complete task with form data → form renders dynamically from `form_schema: FormField[]`
- Reject task with comment
- Delegate task to another role
- SLA breach indicator shown for overdue tasks

### 3.12 Notifications (`/notifications`)

**File**: `src/__tests__/integration/notifications/notification-crud.test.tsx`

Test cases:
- List grouped by date (today, yesterday, this week, older)
- Category tabs: all, security, data, workflow, system
- Mark single as read → verify visual change
- Mark all as read → verify API call + UI update
- Delete notification
- Click notification with action_url → verify navigation

### 3.13 Files (`/files`)

**File**: `src/__tests__/integration/files/file-crud.test.tsx`

Test cases:
- List files with columns: name, type, size, uploaded_by, date
- Upload file: drag-and-drop or click → verify progress bar → success toast
- Download file → verify presigned URL flow
- Delete file with confirmation
- Quarantined files section: list, release, delete

### 3.14 Settings (`/settings`)

**File**: `src/__tests__/integration/settings/profile-settings.test.tsx`

Test cases:
- Update profile: first_name, last_name, email
- Change password: old password, new password (with strength meter)
- Enable MFA: QR code shown → enter TOTP → verify backup codes shown
- Disable MFA: confirm dialog → enter password → verify disabled
- Notification preferences: toggle channels per notification type

### 3.15 Enterprise Suites (Acta, Lex, Visus)

For each suite, create basic CRUD tests for the primary entity:
- **Acta**: Committee CRUD, Meeting CRUD, Action Item CRUD
- **Lex**: Contract CRUD, Document CRUD
- **Visus**: Report CRUD, KPI configuration, Alert CRUD

## PHASE 4: REAL-TIME & WEBSOCKET TESTS

### 4.1 WebSocket Connection (`src/__tests__/integration/websocket.test.tsx`)

Test cases:
- Connect → verify `notification-store.connectionStatus` = "connected"
- Receive `notification.new` → verify notification added to store
- Receive `unread.count` → verify count updated
- Disconnect → verify reconnect with exponential backoff
- Auth token refresh during WebSocket connection

### 4.2 Real-Time Data Updates

Test cases:
- Component using `useRealtimeData` → WebSocket publishes to topic → verify react-query invalidated → data refreshes
- Connection status banner shows when disconnected

## PHASE 5: ERROR HANDLING TESTS

For every page/component that makes API calls:

### 5.1 Network Errors

- Mock `api.ts` to throw network error → verify error state shown (not crash)

### 5.2 401 Unauthorized

- Mock 401 → verify token refresh attempted → verify retry

### 5.3 403 Forbidden

- Mock 403 → verify "insufficient permissions" message

### 5.4 404 Not Found

- Mock 404 on detail page → verify "not found" state

### 5.5 422 Validation Errors

- Mock 422 with field errors → verify errors displayed on correct form fields

### 5.6 429 Rate Limited

- Mock 429 with Retry-After → verify rate limit message shown

### 5.7 500 Server Error

- Mock 500 → verify generic error state, not raw error exposed

## PHASE 6: SECURITY TESTS

### 6.1 CSRF Token

- Verify all state-changing requests include `X-CSRF-Token` header (from `use-csrf` hook)
- Verify CSRF token refreshed on 403

### 6.2 XSS Prevention

- Render notification with `<script>alert('xss')</script>` in title → verify escaped
- Render user name with HTML tags → verify not rendered as HTML
- Verify `sanitizeHtml()` and `sanitizeUrl()` from `lib/sanitize.ts` work

### 6.3 Auth Token Security

- Verify access token NOT in localStorage or sessionStorage
- Verify access token NOT in cookies (only in memory)
- Verify refresh token only in httpOnly cookie (not accessible from JS)

### 6.4 Permission Gates

- Render `<PermissionGate permission="admin:users:create">` without permission → verify nothing rendered
- Render with permission → verify children rendered
- `<PermissionRedirect>` → verify redirect to home when permission missing

## PHASE 7: COMPONENT-LEVEL TESTS

### 7.1 DataTable (`components/shared/data-table.tsx`)

- Renders correct columns
- Sorting: click column header → verify `onSortChange` called
- Pagination: verify page navigation, per_page selector
- Empty state: no data → verify empty message shown
- Loading state: verify skeleton shown
- Error state: verify error message shown
- Row actions: verify dropdown menu works

### 7.2 FormField (`components/shared/forms/form-field.tsx`)

- Renders label, input, error message
- Zod validation: submit invalid → verify error shown
- Required field: submit empty → verify "required" error

### 7.3 Charts (`components/shared/charts/`)

- BarChart: renders with data → verify SVG elements
- PieChart: renders with data → verify arcs
- GaugeChart: renders with value → verify percentage display
- ChartContainer: responsive wrapper → verify resize handling

### 7.4 SeverityIndicator (`components/shared/severity-indicator.tsx`)

- Renders correct color for each severity level: critical, high, medium, low, info

### 7.5 StatusBadge (`components/shared/status-badge.tsx`)

- Renders correct variant for each status

## PHASE 8: STORE TESTS

### 8.1 Auth Store (`stores/auth-store.ts`)

- `login()` → sets user, tenant, isAuthenticated
- `logout()` → clears all state, calls API
- `hasPermission("admin:users:create")` → returns true when user has permission
- `hasPermission("admin:*")` → wildcard matching works
- `hasSuiteAccess("cyber")` → checks suite permissions
- `refreshSession()` → calls BFF, updates user/tenant

### 8.2 Notification Store (`stores/notification-store.ts`)

- `addNotification()` → increments unread count
- `markAsRead()` → decrements unread count
- `markAllAsRead()` → sets unread count to 0
- `setConnectionStatus()` → updates status

### 8.3 Realtime Store (`stores/realtime-store.ts`)

- `register(topic, queryKey)` → associates key with topic
- `publish(topic)` → all registered keys notified
- `unregister()` → removes association

## PHASE 9: BUG FIXING

For EVERY test failure:
1. Read the failing test and source component/hook/store
2. Identify root cause with absolute certainty
3. Fix the source code (not the test) unless the test has a bug
4. Re-run specific test → verify fix
5. Re-run full suite → verify no regressions

Report every bug with: file path, line, what was wrong, what the fix was, before/after.

## PHASE 10: FINAL VERIFICATION

```bash
cd /Users/mac/clario360/frontend
npm run type-check    # Zero TypeScript errors
npm run build         # All 72 pages build successfully
npm run test          # All tests pass
```

Report:
- Total tests: before → after
- New test files created (list each with test count)
- Bugs found and fixed (list each)
- Build output: pages built, bundle size
- Any architectural concerns discovered
