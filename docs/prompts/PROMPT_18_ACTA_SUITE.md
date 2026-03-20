# PROMPT 18 — Acta Suite: Board Governance, Meetings, Minutes, Compliance

## Objective

Complete the Acta (Governance) Suite frontend. Currently only basic list pages exist for committees, meetings, and action items. The backend provides a full board governance platform with meeting lifecycle, agenda management, attendance tracking, minutes authoring/approval, action item tracking, and compliance checks — none of which are implemented in the frontend.

## Reference Architecture

- **Stack**: Next.js 14 App Router, TypeScript, Zustand, React Query, Zod, react-hook-form, Tailwind, shadcn/ui, Recharts
- **Existing types**: `types/suites.ts` — already has comprehensive Acta types (use as-is)
- **API helpers**: `apiGet`, `apiPost`, `apiPut`, `apiPatch`, `apiDelete` from `lib/api.ts`
- **Permissions**: `<PermissionRedirect permission="acta:read">` (or `acta:write` for mutations)
- **WebSocket topics**: `acta.meeting.created`, `acta.meeting.updated`, `acta.meeting.started`, `acta.minutes.submitted`, `acta.minutes.approved`, `acta.action_item.created`, `acta.action_item.overdue`, `acta.compliance.alert`

## Constants to Add (`lib/constants.ts`)

```typescript
// Acta — Dashboard
ACTA_DASHBOARD: '/api/v1/acta/dashboard',

// Acta — Committees (already has ACTA_COMMITTEES)
// Uses: GET, POST /committees, GET/PUT/DELETE /committees/{id}
// Members: POST/PUT/DELETE /committees/{id}/members/{userId}

// Acta — Meetings (already has ACTA_MEETINGS)
ACTA_MEETINGS_UPCOMING: '/api/v1/acta/meetings/upcoming',
ACTA_MEETINGS_CALENDAR: '/api/v1/acta/meetings/calendar',
// Uses: GET/POST /meetings, GET/PUT/DELETE /meetings/{id}
// Lifecycle: POST /meetings/{id}/start, /end, /postpone
// Attendance: GET/POST /meetings/{id}/attendance, POST /meetings/{id}/attendance/bulk
// Attachments: POST/GET /meetings/{id}/attachments, DELETE /meetings/{id}/attachments/{fileId}
// Agenda: POST/GET /meetings/{id}/agenda, PUT/DELETE /meetings/{id}/agenda/{itemId}
//         PUT /meetings/{id}/agenda/reorder, PUT /meetings/{id}/agenda/{itemId}/notes
//         POST /meetings/{id}/agenda/{itemId}/vote
// Minutes: POST/GET/PUT /meetings/{id}/minutes
//          POST /meetings/{id}/minutes/generate, /submit, /request-revision, /approve, /publish
//          GET /meetings/{id}/minutes/versions

// Acta — Action Items (already has ACTA_ACTION_ITEMS)
ACTA_ACTION_ITEMS_OVERDUE: '/api/v1/acta/action-items/overdue',
ACTA_ACTION_ITEMS_MY: '/api/v1/acta/action-items/my',
ACTA_ACTION_ITEMS_STATS: '/api/v1/acta/action-items/stats',
// Uses: POST/GET/PUT /action-items/{id}, PUT /action-items/{id}/status
//       POST /action-items/{id}/extend

// Acta — Compliance
ACTA_COMPLIANCE_RUN: '/api/v1/acta/compliance/run',
ACTA_COMPLIANCE_RESULTS: '/api/v1/acta/compliance/results',
ACTA_COMPLIANCE_REPORT: '/api/v1/acta/compliance/report',
ACTA_COMPLIANCE_SCORE: '/api/v1/acta/compliance/score',
```

---

## PART A — Dashboard (`acta/page.tsx`)

Replace the current basic dashboard with a full data-driven dashboard fetching from `GET /api/v1/acta/dashboard`.

**Layout:**
1. **KPI Cards Row** (from `ActaKPIs`):
   - Active Committees
   - Upcoming Meetings (30d)
   - Open Action Items (with overdue count as warning badge)
   - Compliance Score (gauge or percentage)
   - Minutes Pending Approval
   - Average Attendance Rate

2. **Two-Column Grid:**
   - Left: **Upcoming Meetings** card → list of `ActaMeetingSummary[]` with date, committee, status, quorum status
   - Left: **Recent Meetings** card → last 5 completed meetings
   - Right: **Overdue Action Items** card → list with assignee, due date, priority badge
   - Right: **Compliance by Committee** → horizontal bar chart of compliance scores

3. **Charts Row:**
   - **Meeting Frequency** → bar chart from `meeting_frequency_chart[]` (monthly counts)
   - **Attendance Rate Trend** → line chart from `attendance_rate_chart[]`
   - **Action Items by Status** → donut chart from `action_items_by_status`
   - **Action Items by Priority** → bar chart from `action_items_by_priority`

4. **Recent Activity** → timeline from `recent_activity[]`

**Components:**
```
components/acta/dashboard/
  acta-kpi-cards.tsx
  upcoming-meetings-card.tsx
  overdue-actions-card.tsx
  compliance-by-committee-chart.tsx
  meeting-frequency-chart.tsx
  attendance-rate-chart.tsx
  action-items-charts.tsx
  acta-activity-timeline.tsx
```

---

## PART B — Committee Management (Full CRUD + Members)

### B1. Committees List Page (`acta/committees/page.tsx`)

Enhance with:
- **Create Committee** button → dialog
- Stats in each row: active members, upcoming meetings, open action items
- Per-row actions: Edit, Manage Members, View, Delete

### B2. Committee Create/Edit Dialog

Form fields (validated with Zod):
- Name (required, max 200 chars)
- Type (select from `ActaCommitteeType`: board, audit, risk, compensation, nomination, executive, governance, ad_hoc)
- Description (textarea)
- Chair (user combobox → `GET /api/v1/users`)
- Vice Chair (optional user combobox)
- Secretary (optional user combobox)
- Meeting Frequency (select from `ActaMeetingFrequency`)
- Quorum Type (radio: percentage | fixed_count)
- Quorum Percentage (number 1-100, shown if type=percentage)
- Quorum Fixed Count (number, shown if type=fixed_count)
- Charter (optional textarea)
- Established Date (date picker)
- Tags (tag input)

### B3. Committee Detail Page (`acta/committees/[id]/page.tsx`)

**Tabs:**
1. **Overview** — Committee info, charter, established date, meeting frequency
2. **Members** — DataTable of members with role, joined_at, active status
   - **Add Member** → `POST /api/v1/acta/committees/{id}/members/{userId}` — Dialog with user combobox + role select
   - **Change Role** → `PUT /api/v1/acta/committees/{id}/members/{userId}` — Inline role dropdown
   - **Remove Member** → `DELETE /api/v1/acta/committees/{id}/members/{userId}` — Confirm dialog
3. **Meetings** — Filtered meeting list for this committee
4. **Action Items** — Filtered action items for this committee
5. **Compliance** — Committee-specific compliance checks

**Components:**
```
components/acta/committees/
  committee-columns.tsx
  committee-create-dialog.tsx
  committee-edit-form.tsx
  committee-member-table.tsx
  committee-add-member-dialog.tsx
  committee-member-role-select.tsx
  committee-overview.tsx
```

---

## PART C — Meeting Management (Full Lifecycle)

### C1. Meetings List Page (`acta/meetings/page.tsx`)

Enhance with:
- **Create Meeting** button
- **Calendar View** toggle → `GET /api/v1/acta/meetings/calendar` — Monthly calendar with meeting dots
- **Upcoming filter** → `GET /api/v1/acta/meetings/upcoming`
- Status filter tabs: draft, scheduled, in_progress, completed, cancelled, postponed
- Committee filter

### C2. Meeting Create Dialog

Form fields:
- Committee (combobox → committees list, required)
- Title (required)
- Description (textarea)
- Scheduled At (datetime picker, required)
- Duration Minutes (number, default 60)
- Location Type (radio: physical, virtual, hybrid)
- Location (text, shown if physical/hybrid)
- Virtual Link (URL input, shown if virtual/hybrid)
- Virtual Platform (select: zoom, teams, meet, webex, other)
- Tags

### C3. Meeting Detail Page (`acta/meetings/[id]/page.tsx`) — MAJOR ENHANCEMENT

This is the most complex page. It should support the full meeting lifecycle.

**Header Actions (status-dependent):**
| Status | Actions |
|---|---|
| `draft` | Edit, Schedule (change to scheduled), Delete |
| `scheduled` | Start Meeting, Postpone, Edit, Cancel |
| `in_progress` | End Meeting, Manage Attendance |
| `completed` | View Minutes, Create Minutes |
| `postponed` | Reschedule (edit date) |

**Tabs:**

#### Tab 1: Overview
- Meeting details, committee, status badge, quorum status
- Location info with virtual link (clickable)
- Attendee count vs quorum required

#### Tab 2: Agenda
Full agenda management:
- **Add Agenda Item** → `POST /api/v1/acta/meetings/{id}/agenda`
  - Form: Title, Description, Presenter (user combobox), Duration (minutes), Category (regular/special/information/decision/discussion/ratification), Requires Vote (toggle), Confidential (toggle)
- **Drag-and-drop reorder** → `PUT /api/v1/acta/meetings/{id}/agenda/reorder` — Send `{ item_ids: string[] }`
- **Edit Item** → `PUT .../agenda/{itemId}`
- **Delete Item** → `DELETE .../agenda/{itemId}`
- **Update Notes** → `PUT .../agenda/{itemId}/notes` — Rich text or textarea
- **Record Vote** → `POST .../agenda/{itemId}/vote` — Dialog with:
  - Vote Type (unanimous, majority, two_thirds, roll_call)
  - Votes For (number)
  - Votes Against (number)
  - Votes Abstained (number)
  - Vote Result (auto-calculated or manual: approved, rejected, deferred, tied)
  - Vote Notes (textarea)
- Agenda items display: ordered list with item_number, title, presenter, duration, status badge, vote result if applicable

#### Tab 3: Attendance
- **Attendee Table** → `GET /api/v1/acta/meetings/{id}/attendance`
  - Columns: Name, Email, Role, Status (invited/confirmed/declined/present/absent/proxy/excused), Check-in Time, Check-out Time, Proxy
  - **Record Attendance** → `POST .../attendance` — Change individual status
  - **Bulk Attendance** → `POST .../attendance/bulk` — Mark all as present/absent
  - **Proxy Assignment** — Dialog with proxy user selection and authorization
- Quorum indicator: present_count / quorum_required with visual gauge

#### Tab 4: Minutes
Minutes lifecycle:
- **Create Minutes** → `POST /api/v1/acta/meetings/{id}/minutes` — Rich text editor
- **Generate from Notes** → `POST .../minutes/generate` — AI-generates minutes from agenda notes
- **Edit Minutes** → `PUT .../minutes` — Rich text editor with current content
- **Version History** → `GET .../minutes/versions` — List of previous versions with diff
- **Submit for Review** → `POST .../minutes/submit`
- **Request Revision** → `POST .../minutes/request-revision` — With review notes
- **Approve** → `POST .../minutes/approve`
- **Publish** → `POST .../minutes/publish`

Minutes display:
- Status badge (draft/review/revision_requested/approved/published)
- Content (rendered markdown or HTML)
- AI Summary (if available)
- Extracted Action Items → `ai_action_items[]` — Show with option to convert to real action items
- Reviewer info, approval info, publish date

#### Tab 5: Attachments
- **Upload Attachment** → `POST /api/v1/acta/meetings/{id}/attachments` — File upload
- **List Attachments** → `GET .../attachments` — Table with name, type, uploaded by, date
- **Delete Attachment** → `DELETE .../attachments/{fileId}` — Confirm dialog
- Download link for each attachment

#### Tab 6: Action Items
- Filtered view of action items linked to this meeting
- **Create Action Item** → from this context, pre-fill meeting_id and committee_id

**Components:**
```
components/acta/meetings/
  meeting-columns.tsx
  meeting-create-dialog.tsx
  meeting-edit-form.tsx
  meeting-lifecycle-actions.tsx     — Status-dependent action buttons
  meeting-calendar-view.tsx         — Monthly calendar grid
  meeting-overview.tsx
  meeting-quorum-indicator.tsx      — Visual quorum gauge

  agenda/
    agenda-list.tsx                 — Ordered, drag-and-drop sortable
    agenda-item-card.tsx            — Single agenda item display
    agenda-create-dialog.tsx
    agenda-edit-dialog.tsx
    agenda-notes-editor.tsx
    agenda-vote-dialog.tsx          — Record vote with tallies
    agenda-vote-result.tsx          — Display vote outcome

  attendance/
    attendance-table.tsx
    attendance-record-dialog.tsx
    attendance-bulk-dialog.tsx
    attendance-proxy-dialog.tsx
    quorum-gauge.tsx

  minutes/
    minutes-editor.tsx              — Rich text editor for content
    minutes-viewer.tsx              — Rendered minutes display
    minutes-generate-button.tsx     — AI generation trigger
    minutes-status-actions.tsx      — Submit/Review/Approve/Publish buttons
    minutes-version-history.tsx     — Version list with diff viewer
    minutes-extracted-actions.tsx   — AI-extracted action items
    minutes-review-notes.tsx

  attachments/
    attachment-upload.tsx
    attachment-list.tsx
```

---

## PART D — Action Item Management (Full CRUD + Tracking)

### D1. Action Items Page (`acta/action-items/page.tsx`)

Enhance with:
- **Create Action Item** button
- **My Action Items** tab → `GET /api/v1/acta/action-items/my`
- **Overdue** tab → `GET /api/v1/acta/action-items/overdue`
- **All** tab → `GET /api/v1/acta/action-items`
- Stats bar → `GET /api/v1/acta/action-items/stats`
- **Kanban View** toggle — Columns by status (pending, in_progress, completed, overdue)
- Filters: committee, priority, assignee, date range

### D2. Action Item Create/Edit Dialog

Form fields:
- Title (required)
- Description (textarea)
- Committee (combobox, required)
- Meeting (optional combobox — filter by committee)
- Agenda Item (optional combobox — filter by meeting)
- Priority (select: critical, high, medium, low)
- Assigned To (user combobox, required)
- Due Date (date picker, required)
- Tags

### D3. Action Item Detail (slide-out panel or dialog)

- Title, description, priority badge, status badge
- Meeting link, committee link
- Assignee info
- Due date with overdue indicator
- Extended count and extension history
- **Status Actions:**
  - Change Status → `PUT /api/v1/acta/action-items/{id}/status` — Select new status
  - Extend Deadline → `POST /api/v1/acta/action-items/{id}/extend` — New date + reason
  - Complete → Status change to completed with completion notes and evidence
- Edit → `PUT /api/v1/acta/action-items/{id}`

**Components:**
```
components/acta/action-items/
  action-item-columns.tsx
  action-item-create-dialog.tsx
  action-item-edit-form.tsx
  action-item-detail-panel.tsx
  action-item-status-dialog.tsx
  action-item-extend-dialog.tsx
  action-item-kanban.tsx           — Kanban board view
  action-item-stats-bar.tsx
```

---

## PART E — Compliance Checks

### E1. Compliance Page (`acta/compliance/page.tsx`)

Full compliance dashboard:
- **Run Compliance Check** button → `GET /api/v1/acta/compliance/run` — Triggers check, returns results
- **Compliance Score** → `GET /api/v1/acta/compliance/score` — Large gauge display
- **Results** → `GET /api/v1/acta/compliance/results` — DataTable with:
  - Columns: check_type, check_name, committee, status (compliant/non_compliant/warning), severity, finding, recommendation
  - Filter by status, severity, committee, check_type
- **Report** → `GET /api/v1/acta/compliance/report` — Full report view with:
  - Overall score
  - By-status breakdown (pie chart)
  - By-check-type breakdown (bar chart)
  - By-committee table with individual scores
  - Detailed findings list

**Components:**
```
components/acta/compliance/
  compliance-run-button.tsx
  compliance-score-gauge.tsx
  compliance-results-table.tsx
  compliance-report-viewer.tsx
  compliance-by-committee-chart.tsx
  compliance-by-type-chart.tsx
```

---

## Sidebar Navigation

Update Acta section:
```
Acta
├── Dashboard          (enhance with full dashboard)
├── Committees         (add CRUD + member management)
├── Meetings           (add full lifecycle, agenda, minutes, attendance)
├── Action Items       (add CRUD + kanban + tracking)
└── Compliance         (add checks, report, score)
```

---

## Testing Requirements

- All new components must have Vitest + React Testing Library tests
- Test meeting lifecycle: create → schedule → start → end → minutes
- Test agenda management: add, reorder, vote
- Test attendance flow: invite → confirm → present, quorum calculation
- Test minutes workflow: create/generate → submit → review → approve → publish
- Test action item lifecycle: create → assign → update status → extend → complete
- Test compliance: run check → view results → view report
- Test calendar view rendering
- Test kanban drag-and-drop (if implemented)

## Validation Checklist

- [ ] `npm run build` passes with zero errors
- [ ] Every backend endpoint under `/api/v1/acta/*` has a corresponding frontend call
- [ ] Meeting lifecycle state machine is correctly enforced (only valid transitions)
- [ ] Quorum calculation displays correctly
- [ ] Minutes version history shows diffs
- [ ] AI-extracted action items can be converted to real action items
- [ ] Calendar view correctly positions meetings
- [ ] Kanban view works with drag-and-drop status changes
- [ ] Attendance proxy assignments work correctly
- [ ] Vote tallies auto-calculate results
