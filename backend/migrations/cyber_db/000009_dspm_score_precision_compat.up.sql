ALTER TABLE dspm_data_assets
    ALTER COLUMN sensitivity_score TYPE NUMERIC(5,2)
    USING CASE
        WHEN sensitivity_score IS NULL THEN 0
        WHEN sensitivity_score <= 1 THEN ROUND((sensitivity_score * 100)::numeric, 2)
        ELSE ROUND(sensitivity_score::numeric, 2)
    END;

ALTER TABLE dspm_data_assets
    ALTER COLUMN risk_score TYPE NUMERIC(5,2)
    USING CASE
        WHEN risk_score IS NULL THEN 0
        WHEN risk_score <= 1 THEN ROUND((risk_score * 100)::numeric, 2)
        ELSE ROUND(risk_score::numeric, 2)
    END;

ALTER TABLE dspm_data_assets
    ALTER COLUMN posture_score TYPE NUMERIC(5,2)
    USING CASE
        WHEN posture_score IS NULL THEN 0
        WHEN posture_score <= 1 THEN ROUND((posture_score * 100)::numeric, 2)
        ELSE ROUND(posture_score::numeric, 2)
    END;
