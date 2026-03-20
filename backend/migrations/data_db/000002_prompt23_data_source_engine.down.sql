DROP TABLE IF EXISTS sync_history;

DROP INDEX IF EXISTS idx_sources_next_sync;
DROP INDEX IF EXISTS idx_sources_tenant_name_unique;
DROP INDEX IF EXISTS idx_models_pii;
DROP INDEX IF EXISTS idx_models_tenant_name_version_unique;
