DROP TABLE IF EXISTS vciso_control_dependencies;
DROP TABLE IF EXISTS vciso_benchmarks;
ALTER TABLE vciso_evidence DROP COLUMN IF EXISTS verified_by;
