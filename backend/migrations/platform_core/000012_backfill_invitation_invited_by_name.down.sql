-- Down migration: there is no safe way to restore the original email values
-- after the backfill because the original strings were never stored elsewhere.
-- The down migration is intentionally a no-op; rolling back this migration
-- will not revert the invited_by_name values.
SELECT 1;
