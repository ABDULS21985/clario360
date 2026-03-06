-- =============================================================================
-- Clario 360 — Acta Suite Database Schema
-- Database: acta_db
-- Contains: committees, meetings, agenda items, minutes, action items,
--           governance workflows, compliance checks
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE committee_status AS ENUM ('active', 'inactive', 'dissolved');
COMMENT ON TYPE committee_status IS 'Lifecycle status of a committee';

CREATE TYPE meeting_status AS ENUM ('scheduled', 'in_progress', 'completed', 'cancelled');
COMMENT ON TYPE meeting_status IS 'Status of a meeting';

CREATE TYPE agenda_item_status AS ENUM ('pending', 'discussed', 'deferred', 'approved', 'rejected');
COMMENT ON TYPE agenda_item_status IS 'Status of an individual agenda item';

CREATE TYPE minutes_status AS ENUM ('draft', 'review', 'approved', 'published');
COMMENT ON TYPE minutes_status IS 'Approval lifecycle of meeting minutes';

CREATE TYPE action_item_status AS ENUM ('pending', 'in_progress', 'completed', 'overdue', 'cancelled');
COMMENT ON TYPE action_item_status IS 'Status of an action item assigned from a meeting';

CREATE TYPE workflow_status AS ENUM ('active', 'inactive', 'archived');
COMMENT ON TYPE workflow_status IS 'Status of a governance workflow definition';

CREATE TYPE workflow_instance_status AS ENUM ('active', 'completed', 'cancelled', 'suspended');
COMMENT ON TYPE workflow_instance_status IS 'Status of a running workflow instance';

CREATE TYPE compliance_check_status AS ENUM ('compliant', 'non_compliant', 'warning', 'not_applicable');
COMMENT ON TYPE compliance_check_status IS 'Result of a compliance check';

-- =============================================================================
-- TRIGGER FUNCTION
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TABLE: committees
-- =============================================================================

CREATE TABLE committees (
    id                UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID             NOT NULL,
    name              VARCHAR(255)     NOT NULL,
    type              VARCHAR(100)     NOT NULL,
    description       TEXT             NOT NULL DEFAULT '',
    chair_user_id     UUID,
    secretary_user_id UUID,
    members           JSONB            NOT NULL DEFAULT '[]',
    meeting_frequency VARCHAR(50),
    status            committee_status NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID
);

COMMENT ON TABLE committees IS 'Board committees and governance bodies';
COMMENT ON COLUMN committees.chair_user_id IS 'User chairing the committee (references platform_core.users)';
COMMENT ON COLUMN committees.secretary_user_id IS 'User serving as secretary (references platform_core.users)';
COMMENT ON COLUMN committees.members IS 'JSON array of member objects (user_id, role, joined_at)';
COMMENT ON COLUMN committees.meeting_frequency IS 'How often the committee meets (e.g., weekly, monthly)';

CREATE INDEX idx_committees_tenant_status ON committees (tenant_id, status);
CREATE INDEX idx_committees_tenant_created ON committees (tenant_id, created_at DESC);
CREATE INDEX idx_committees_members ON committees USING GIN (members);

CREATE TRIGGER trg_committees_updated_at
    BEFORE UPDATE ON committees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: meetings
-- =============================================================================

CREATE TABLE meetings (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID           NOT NULL,
    committee_id     UUID           NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    title            VARCHAR(500)   NOT NULL,
    description      TEXT           NOT NULL DEFAULT '',
    scheduled_at     TIMESTAMPTZ    NOT NULL,
    location         VARCHAR(500),
    virtual_link     TEXT,
    status           meeting_status NOT NULL DEFAULT 'scheduled',
    duration_minutes INTEGER,
    created_by       UUID,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_by       UUID
);

COMMENT ON TABLE meetings IS 'Scheduled and completed committee meetings';
COMMENT ON COLUMN meetings.virtual_link IS 'Video conferencing link for remote meetings';
COMMENT ON COLUMN meetings.duration_minutes IS 'Actual or planned duration in minutes';

CREATE INDEX idx_meetings_tenant_status ON meetings (tenant_id, status);
CREATE INDEX idx_meetings_committee ON meetings (committee_id, scheduled_at DESC);
CREATE INDEX idx_meetings_scheduled ON meetings (scheduled_at);
CREATE INDEX idx_meetings_tenant_created ON meetings (tenant_id, created_at DESC);

CREATE TRIGGER trg_meetings_updated_at
    BEFORE UPDATE ON meetings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: agenda_items
-- =============================================================================

CREATE TABLE agenda_items (
    id                UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID               NOT NULL,
    meeting_id        UUID               NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    title             VARCHAR(500)       NOT NULL,
    description       TEXT               NOT NULL DEFAULT '',
    presenter_user_id UUID,
    duration_minutes  INTEGER,
    order_index       INTEGER            NOT NULL DEFAULT 0,
    status            agenda_item_status NOT NULL DEFAULT 'pending',
    attachments       JSONB              DEFAULT '[]',
    voting_result     JSONB,
    created_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID
);

COMMENT ON TABLE agenda_items IS 'Individual items on a meeting agenda';
COMMENT ON COLUMN agenda_items.presenter_user_id IS 'User presenting this item (references platform_core.users)';
COMMENT ON COLUMN agenda_items.order_index IS 'Display order within the meeting agenda';
COMMENT ON COLUMN agenda_items.attachments IS 'JSON array of attachment references (name, url, type)';
COMMENT ON COLUMN agenda_items.voting_result IS 'Voting outcome if item was put to a vote (for, against, abstain)';

CREATE INDEX idx_agenda_meeting ON agenda_items (meeting_id, order_index);
CREATE INDEX idx_agenda_tenant_status ON agenda_items (tenant_id, status);
CREATE INDEX idx_agenda_attachments ON agenda_items USING GIN (attachments);
CREATE INDEX idx_agenda_voting ON agenda_items USING GIN (voting_result) WHERE voting_result IS NOT NULL;

CREATE TRIGGER trg_agenda_items_updated_at
    BEFORE UPDATE ON agenda_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: meeting_minutes
-- =============================================================================

CREATE TABLE meeting_minutes (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID           NOT NULL,
    meeting_id      UUID           NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    content         TEXT           NOT NULL DEFAULT '',
    ai_summary      TEXT,
    ai_action_items JSONB          DEFAULT '[]',
    status          minutes_status NOT NULL DEFAULT 'draft',
    approved_by     UUID,
    approved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_by      UUID,
    updated_by      UUID,
    CONSTRAINT uq_minutes_meeting UNIQUE (meeting_id)
);

COMMENT ON TABLE meeting_minutes IS 'Official minutes of a meeting with AI-generated summaries';
COMMENT ON COLUMN meeting_minutes.ai_summary IS 'AI-generated summary of the meeting';
COMMENT ON COLUMN meeting_minutes.ai_action_items IS 'AI-extracted action items from the minutes';
COMMENT ON COLUMN meeting_minutes.approved_by IS 'User who approved the minutes (references platform_core.users)';

CREATE INDEX idx_minutes_tenant_status ON meeting_minutes (tenant_id, status);
CREATE INDEX idx_minutes_meeting ON meeting_minutes (meeting_id);
CREATE INDEX idx_minutes_ai_actions ON meeting_minutes USING GIN (ai_action_items);

CREATE TRIGGER trg_minutes_updated_at
    BEFORE UPDATE ON meeting_minutes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: action_items
-- =============================================================================

CREATE TABLE action_items (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID               NOT NULL,
    meeting_id     UUID               NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    agenda_item_id UUID               REFERENCES agenda_items(id) ON DELETE SET NULL,
    title          VARCHAR(500)       NOT NULL,
    description    TEXT               NOT NULL DEFAULT '',
    assigned_to    UUID,
    due_date       DATE,
    status         action_item_status NOT NULL DEFAULT 'pending',
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    created_by     UUID,
    updated_by     UUID
);

COMMENT ON TABLE action_items IS 'Action items arising from meetings, tracked to completion';
COMMENT ON COLUMN action_items.assigned_to IS 'User responsible for this action (references platform_core.users)';

CREATE INDEX idx_actions_tenant_status ON action_items (tenant_id, status);
CREATE INDEX idx_actions_meeting ON action_items (meeting_id);
CREATE INDEX idx_actions_assigned ON action_items (assigned_to) WHERE status NOT IN ('completed', 'cancelled');
CREATE INDEX idx_actions_due_date ON action_items (due_date) WHERE status NOT IN ('completed', 'cancelled');
CREATE INDEX idx_actions_overdue ON action_items (tenant_id, due_date)
    WHERE status NOT IN ('completed', 'cancelled') AND due_date < NOW();
CREATE INDEX idx_actions_tenant_created ON action_items (tenant_id, created_at DESC);

CREATE TRIGGER trg_action_items_updated_at
    BEFORE UPDATE ON action_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: governance_workflows
-- =============================================================================

CREATE TABLE governance_workflows (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID            NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    type        VARCHAR(100)    NOT NULL,
    definition  JSONB           NOT NULL DEFAULT '{}',
    status      workflow_status NOT NULL DEFAULT 'active',
    created_by  UUID,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_by  UUID
);

COMMENT ON TABLE governance_workflows IS 'BPMN-like workflow definitions for governance processes';
COMMENT ON COLUMN governance_workflows.definition IS 'Workflow definition (steps, transitions, conditions)';

CREATE INDEX idx_gov_wf_tenant_status ON governance_workflows (tenant_id, status);
CREATE INDEX idx_gov_wf_type ON governance_workflows (tenant_id, type);
CREATE INDEX idx_gov_wf_definition ON governance_workflows USING GIN (definition);

CREATE TRIGGER trg_gov_workflows_updated_at
    BEFORE UPDATE ON governance_workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: workflow_instances
-- =============================================================================

CREATE TABLE workflow_instances (
    id           UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID                     NOT NULL,
    workflow_id  UUID                     NOT NULL REFERENCES governance_workflows(id) ON DELETE CASCADE,
    current_step VARCHAR(100)             NOT NULL DEFAULT '',
    data         JSONB                    NOT NULL DEFAULT '{}',
    status       workflow_instance_status NOT NULL DEFAULT 'active',
    started_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    created_by   UUID,
    updated_by   UUID
);

COMMENT ON TABLE workflow_instances IS 'Running instances of governance workflows';
COMMENT ON COLUMN workflow_instances.current_step IS 'Identifier of the current step in the workflow';
COMMENT ON COLUMN workflow_instances.data IS 'Instance-specific data accumulated during execution';

CREATE INDEX idx_wf_inst_tenant_status ON workflow_instances (tenant_id, status);
CREATE INDEX idx_wf_inst_workflow ON workflow_instances (workflow_id);
CREATE INDEX idx_wf_inst_data ON workflow_instances USING GIN (data);

CREATE TRIGGER trg_wf_instances_updated_at
    BEFORE UPDATE ON workflow_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: compliance_checks
-- =============================================================================

CREATE TABLE compliance_checks (
    id          UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID                     NOT NULL,
    entity_type VARCHAR(100)             NOT NULL,
    entity_id   UUID                     NOT NULL,
    rule_name   VARCHAR(255)             NOT NULL,
    status      compliance_check_status  NOT NULL,
    details     JSONB                    DEFAULT '{}',
    checked_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    created_by  UUID
);

COMMENT ON TABLE compliance_checks IS 'Compliance check results against governance rules';
COMMENT ON COLUMN compliance_checks.entity_type IS 'Type of entity checked (e.g., meeting, workflow, document)';
COMMENT ON COLUMN compliance_checks.entity_id IS 'UUID of the checked entity';
COMMENT ON COLUMN compliance_checks.rule_name IS 'Name of the compliance rule applied';
COMMENT ON COLUMN compliance_checks.details IS 'Detailed check results and findings';

CREATE INDEX idx_compliance_tenant_status ON compliance_checks (tenant_id, status);
CREATE INDEX idx_compliance_entity ON compliance_checks (entity_type, entity_id);
CREATE INDEX idx_compliance_rule ON compliance_checks (rule_name);
CREATE INDEX idx_compliance_checked ON compliance_checks (checked_at DESC);
CREATE INDEX idx_compliance_details ON compliance_checks USING GIN (details);
