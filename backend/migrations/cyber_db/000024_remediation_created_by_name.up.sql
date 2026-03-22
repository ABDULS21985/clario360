-- Add created_by_name to remediation_actions so the UI can display the creator's name
-- without a cross-service JOIN to the IAM users table.
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS created_by_name TEXT;
