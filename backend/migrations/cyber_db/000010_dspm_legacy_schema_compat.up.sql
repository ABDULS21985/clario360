UPDATE dspm_data_assets da
SET
    name = COALESCE(NULLIF(da.name, ''), a.name),
    type = COALESCE(NULLIF(da.type, ''), a.type::text),
    location = COALESCE(NULLIF(da.location, ''), a.location, ''),
    classification = COALESCE(da.classification, da.data_classification::data_classification)
FROM assets a
WHERE da.asset_id = a.id;

ALTER TABLE dspm_data_assets
    ALTER COLUMN name DROP NOT NULL,
    ALTER COLUMN type DROP NOT NULL,
    ALTER COLUMN location DROP NOT NULL;

ALTER TABLE dspm_data_assets
    DROP CONSTRAINT IF EXISTS dspm_data_assets_risk_score_check,
    DROP CONSTRAINT IF EXISTS dspm_data_assets_sensitivity_score_check;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_dspm_data_assets_risk_score_prompt20'
    ) THEN
        ALTER TABLE dspm_data_assets
            ADD CONSTRAINT chk_dspm_data_assets_risk_score_prompt20
            CHECK (risk_score >= 0 AND risk_score <= 100);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_dspm_data_assets_sensitivity_score_prompt20'
    ) THEN
        ALTER TABLE dspm_data_assets
            ADD CONSTRAINT chk_dspm_data_assets_sensitivity_score_prompt20
            CHECK (sensitivity_score >= 0 AND sensitivity_score <= 100);
    END IF;
END $$;
