CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS committees (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    name                TEXT            NOT NULL,
    type                TEXT            NOT NULL CHECK (type IN (
        'board', 'audit', 'risk', 'compensation', 'nomination',
        'executive', 'governance', 'ad_hoc'
    )),
    description         TEXT            NOT NULL DEFAULT '',
    chair_user_id       UUID            NOT NULL,
    vice_chair_user_id  UUID,
    secretary_user_id   UUID,
    meeting_frequency   TEXT            NOT NULL DEFAULT 'monthly'
                                        CHECK (meeting_frequency IN (
        'weekly', 'bi_weekly', 'monthly', 'quarterly', 'semi_annual', 'annual', 'ad_hoc'
    )),
    quorum_percentage   INT             NOT NULL DEFAULT 51 CHECK (quorum_percentage BETWEEN 1 AND 100),
    quorum_type         TEXT            NOT NULL DEFAULT 'percentage'
                                        CHECK (quorum_type IN ('percentage', 'fixed_count')),
    quorum_fixed_count  INT,
    charter             TEXT,
    established_date    DATE,
    dissolution_date    DATE,
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'inactive', 'dissolved')),
    tags                TEXT[]          NOT NULL DEFAULT '{}',
    metadata            JSONB           NOT NULL DEFAULT '{}',
    created_by          UUID            NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT committees_fixed_quorum_required CHECK (
        (quorum_type = 'percentage')
        OR (quorum_type = 'fixed_count' AND quorum_fixed_count IS NOT NULL AND quorum_fixed_count > 0)
    ),
    CONSTRAINT committees_dissolution_after_established CHECK (
        dissolution_date IS NULL
        OR established_date IS NULL
        OR dissolution_date >= established_date
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_committees_tenant_name_unique
    ON committees (tenant_id, name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_committees_tenant
    ON committees (tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS committee_members (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    committee_id    UUID            NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    user_id         UUID            NOT NULL,
    user_name       TEXT            NOT NULL,
    user_email      TEXT            NOT NULL,
    role            TEXT            NOT NULL DEFAULT 'member'
                                    CHECK (role IN ('chair', 'vice_chair', 'secretary', 'member', 'observer')),
    joined_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    left_at         TIMESTAMPTZ,
    active          BOOLEAN         NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT committee_members_leave_after_join CHECK (left_at IS NULL OR left_at >= joined_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_committee_members_committee_user_active_unique
    ON committee_members (committee_id, user_id)
    WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_committee_members_committee
    ON committee_members (committee_id)
    WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_committee_members_user
    ON committee_members (user_id)
    WHERE active = true;

CREATE TABLE IF NOT EXISTS meetings (
    id                   UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID            NOT NULL,
    committee_id         UUID            NOT NULL REFERENCES committees(id),
    committee_name       TEXT            NOT NULL,
    title                TEXT            NOT NULL,
    description          TEXT            NOT NULL DEFAULT '',
    meeting_number       INT,
    scheduled_at         TIMESTAMPTZ     NOT NULL,
    scheduled_end_at     TIMESTAMPTZ,
    actual_start_at      TIMESTAMPTZ,
    actual_end_at        TIMESTAMPTZ,
    duration_minutes     INT             NOT NULL DEFAULT 60 CHECK (duration_minutes BETWEEN 15 AND 480),
    location             TEXT,
    location_type        TEXT            NOT NULL DEFAULT 'physical'
                                         CHECK (location_type IN ('physical', 'virtual', 'hybrid')),
    virtual_link         TEXT,
    virtual_platform     TEXT,
    status               TEXT            NOT NULL DEFAULT 'scheduled'
                                         CHECK (status IN ('draft', 'scheduled', 'in_progress', 'completed', 'cancelled', 'postponed')),
    cancellation_reason  TEXT,
    quorum_required      INT             NOT NULL CHECK (quorum_required > 0),
    attendee_count       INT             NOT NULL DEFAULT 0 CHECK (attendee_count >= 0),
    present_count        INT             NOT NULL DEFAULT 0 CHECK (present_count >= 0),
    quorum_met           BOOLEAN,
    agenda_item_count    INT             NOT NULL DEFAULT 0 CHECK (agenda_item_count >= 0),
    action_item_count    INT             NOT NULL DEFAULT 0 CHECK (action_item_count >= 0),
    has_minutes          BOOLEAN         NOT NULL DEFAULT false,
    minutes_status       TEXT            CHECK (minutes_status IN ('draft', 'review', 'revision_requested', 'approved', 'published')),
    workflow_instance_id UUID,
    tags                 TEXT[]          NOT NULL DEFAULT '{}',
    metadata             JSONB           NOT NULL DEFAULT '{}',
    created_by           UUID            NOT NULL,
    created_at           TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ,
    CONSTRAINT meetings_schedule_window CHECK (
        scheduled_end_at IS NULL OR scheduled_end_at >= scheduled_at
    ),
    CONSTRAINT meetings_actual_window CHECK (
        actual_end_at IS NULL OR actual_start_at IS NULL OR actual_end_at >= actual_start_at
    )
);

CREATE INDEX IF NOT EXISTS idx_meetings_tenant_status
    ON meetings (tenant_id, status, scheduled_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meetings_committee
    ON meetings (committee_id, scheduled_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meetings_upcoming
    ON meetings (tenant_id, scheduled_at)
    WHERE status IN ('scheduled', 'draft') AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meetings_calendar
    ON meetings (tenant_id, date_trunc('month', scheduled_at AT TIME ZONE 'UTC'))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS meeting_attendance (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    meeting_id          UUID            NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id             UUID            NOT NULL,
    user_name           TEXT            NOT NULL,
    user_email          TEXT            NOT NULL,
    member_role         TEXT            NOT NULL,
    status              TEXT            NOT NULL DEFAULT 'invited'
                                        CHECK (status IN ('invited', 'confirmed', 'declined', 'present', 'absent', 'proxy', 'excused')),
    confirmed_at        TIMESTAMPTZ,
    checked_in_at       TIMESTAMPTZ,
    checked_out_at      TIMESTAMPTZ,
    proxy_user_id       UUID,
    proxy_user_name     TEXT,
    proxy_authorized_by UUID,
    notes               TEXT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT meeting_attendance_proxy_requires_name CHECK (
        proxy_user_id IS NULL OR proxy_user_name IS NOT NULL
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meeting_attendance_meeting_user_unique
    ON meeting_attendance (meeting_id, user_id);

CREATE INDEX IF NOT EXISTS idx_attendance_meeting
    ON meeting_attendance (meeting_id, status);

CREATE INDEX IF NOT EXISTS idx_attendance_user
    ON meeting_attendance (user_id, meeting_id);

CREATE TABLE IF NOT EXISTS agenda_items (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    meeting_id          UUID            NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    title               TEXT            NOT NULL,
    description         TEXT            NOT NULL DEFAULT '',
    item_number         TEXT,
    presenter_user_id   UUID,
    presenter_name      TEXT,
    duration_minutes    INT             NOT NULL DEFAULT 15 CHECK (duration_minutes BETWEEN 1 AND 480),
    order_index         INT             NOT NULL CHECK (order_index >= 0),
    parent_item_id      UUID            REFERENCES agenda_items(id),
    status              TEXT            NOT NULL DEFAULT 'pending'
                                        CHECK (status IN ('pending', 'discussed', 'deferred', 'approved', 'rejected', 'withdrawn', 'for_noting')),
    notes               TEXT,
    requires_vote       BOOLEAN         NOT NULL DEFAULT false,
    vote_type           TEXT            CHECK (vote_type IN ('unanimous', 'majority', 'two_thirds', 'roll_call')),
    votes_for           INT             CHECK (votes_for IS NULL OR votes_for >= 0),
    votes_against       INT             CHECK (votes_against IS NULL OR votes_against >= 0),
    votes_abstained     INT             CHECK (votes_abstained IS NULL OR votes_abstained >= 0),
    vote_result         TEXT            CHECK (vote_result IN ('approved', 'rejected', 'deferred', 'tied')),
    vote_notes          TEXT,
    attachment_ids      UUID[]          NOT NULL DEFAULT '{}',
    category            TEXT            CHECK (category IN (
        'regular', 'special', 'information', 'decision', 'discussion', 'ratification'
    )),
    confidential        BOOLEAN         NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT agenda_items_vote_consistency CHECK (
        requires_vote = false
        OR vote_type IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_agenda_meeting
    ON agenda_items (meeting_id, order_index);

CREATE INDEX IF NOT EXISTS idx_agenda_parent
    ON agenda_items (parent_item_id)
    WHERE parent_item_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS meeting_minutes (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL,
    meeting_id              UUID            NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    content                 TEXT            NOT NULL,
    ai_summary              TEXT,
    status                  TEXT            NOT NULL DEFAULT 'draft'
                                            CHECK (status IN ('draft', 'review', 'revision_requested', 'approved', 'published')),
    submitted_for_review_at TIMESTAMPTZ,
    submitted_by            UUID,
    reviewed_by             UUID,
    review_notes            TEXT,
    approved_by             UUID,
    approved_at             TIMESTAMPTZ,
    published_at            TIMESTAMPTZ,
    version                 INT             NOT NULL DEFAULT 1 CHECK (version > 0),
    previous_version_id     UUID            REFERENCES meeting_minutes(id),
    ai_action_items         JSONB           NOT NULL DEFAULT '[]',
    ai_generated            BOOLEAN         NOT NULL DEFAULT false,
    created_by              UUID            NOT NULL,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_minutes_meeting_version_unique
    ON meeting_minutes (meeting_id, version);

CREATE INDEX IF NOT EXISTS idx_minutes_meeting
    ON meeting_minutes (meeting_id, version DESC);

CREATE INDEX IF NOT EXISTS idx_minutes_status
    ON meeting_minutes (tenant_id, status);

CREATE TABLE IF NOT EXISTS action_items (
    id                   UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID            NOT NULL,
    meeting_id           UUID            NOT NULL REFERENCES meetings(id),
    agenda_item_id       UUID            REFERENCES agenda_items(id),
    committee_id         UUID            NOT NULL REFERENCES committees(id),
    title                TEXT            NOT NULL,
    description          TEXT            NOT NULL DEFAULT '',
    priority             TEXT            NOT NULL DEFAULT 'medium'
                                         CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    assigned_to          UUID            NOT NULL,
    assignee_name        TEXT            NOT NULL,
    assigned_by          UUID            NOT NULL,
    due_date             DATE            NOT NULL,
    original_due_date    DATE            NOT NULL,
    extended_count       INT             NOT NULL DEFAULT 0 CHECK (extended_count >= 0),
    extension_reason     TEXT,
    status               TEXT            NOT NULL DEFAULT 'pending'
                                         CHECK (status IN ('pending', 'in_progress', 'completed', 'overdue', 'cancelled', 'deferred')),
    completed_at         TIMESTAMPTZ,
    completion_notes     TEXT,
    completion_evidence  UUID[]          NOT NULL DEFAULT '{}',
    follow_up_meeting_id UUID            REFERENCES meetings(id),
    reviewed_at          TIMESTAMPTZ,
    tags                 TEXT[]          NOT NULL DEFAULT '{}',
    metadata             JSONB           NOT NULL DEFAULT '{}',
    created_by           UUID            NOT NULL,
    created_at           TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT action_items_due_extension CHECK (due_date >= original_due_date OR extended_count > 0)
);

CREATE INDEX IF NOT EXISTS idx_action_items_meeting
    ON action_items (meeting_id);

CREATE INDEX IF NOT EXISTS idx_action_items_committee
    ON action_items (committee_id, status);

CREATE INDEX IF NOT EXISTS idx_action_items_assignee
    ON action_items (assigned_to, status);

CREATE INDEX IF NOT EXISTS idx_action_items_overdue
    ON action_items (tenant_id, due_date)
    WHERE status IN ('pending', 'in_progress', 'overdue');

CREATE INDEX IF NOT EXISTS idx_action_items_tenant
    ON action_items (tenant_id, status, due_date);

CREATE TABLE IF NOT EXISTS compliance_checks (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    committee_id    UUID            REFERENCES committees(id),
    check_type      TEXT            NOT NULL CHECK (check_type IN (
        'meeting_frequency', 'quorum_compliance', 'minutes_completion',
        'action_item_tracking', 'attendance_rate', 'charter_review',
        'document_retention', 'conflict_of_interest'
    )),
    check_name      TEXT            NOT NULL,
    status          TEXT            NOT NULL CHECK (status IN ('compliant', 'non_compliant', 'warning', 'not_applicable')),
    severity        TEXT            NOT NULL DEFAULT 'medium'
                                    CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    description     TEXT            NOT NULL,
    finding         TEXT,
    recommendation  TEXT,
    evidence        JSONB           NOT NULL DEFAULT '{}',
    period_start    DATE            NOT NULL,
    period_end      DATE            NOT NULL,
    checked_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    checked_by      TEXT            NOT NULL DEFAULT 'system',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT compliance_period_window CHECK (period_end >= period_start)
);

CREATE INDEX IF NOT EXISTS idx_compliance_tenant
    ON compliance_checks (tenant_id, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_committee
    ON compliance_checks (committee_id, check_type, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_status
    ON compliance_checks (tenant_id, status)
    WHERE status IN ('non_compliant', 'warning');

CREATE TRIGGER trg_committees_updated_at
    BEFORE UPDATE ON committees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_committee_members_updated_at
    BEFORE UPDATE ON committee_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_meetings_updated_at
    BEFORE UPDATE ON meetings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_meeting_attendance_updated_at
    BEFORE UPDATE ON meeting_attendance
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_agenda_items_updated_at
    BEFORE UPDATE ON agenda_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_meeting_minutes_updated_at
    BEFORE UPDATE ON meeting_minutes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_action_items_updated_at
    BEFORE UPDATE ON action_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
