DROP TRIGGER IF EXISTS trg_invitations_updated_at ON invitations;
DROP TABLE IF EXISTS invitations;

DROP TABLE IF EXISTS email_verifications;
DROP TABLE IF EXISTS provisioning_steps;

DROP TRIGGER IF EXISTS trg_tenant_onboarding_updated_at ON tenant_onboarding;
DROP TABLE IF EXISTS tenant_onboarding;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS retain_until,
    DROP COLUMN IF EXISTS deprovisioned_by,
    DROP COLUMN IF EXISTS deprovisioned_at;
