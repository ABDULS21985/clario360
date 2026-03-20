-- Backfill invited_by_name on the invitations table.
--
-- Historically the IAM service stored the inviting user's email address in
-- invited_by_name because the JWT ContextUser struct only carried an email
-- field.  The service now resolves the inviter's full name via a userRepo
-- lookup before persisting the invitation, so going forward names will be
-- stored correctly.  This migration fixes already-persisted rows by joining
-- against the users table on the invited_by UUID.
--
-- Rules:
--   • Only rows where invited_by_name looks like an email address (contains
--     '@') are candidates for backfill – rows that already hold a proper
--     display name are left untouched.
--   • The new value is TRIM(first_name || ' ' || last_name).  If the join
--     finds no matching user the row is left unchanged.
--   • invited_by is a UUID foreign key; NULL rows (should not exist) are
--     skipped automatically by the JOIN.

UPDATE invitations AS i
SET    invited_by_name = TRIM(u.first_name || ' ' || u.last_name),
       updated_at      = now()
FROM   users AS u
WHERE  i.invited_by  = u.id
  AND  i.invited_by_name LIKE '%@%'           -- looks like an email
  AND  TRIM(u.first_name || ' ' || u.last_name) <> '';  -- user has a real name
