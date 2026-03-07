DROP TRIGGER IF EXISTS trg_action_items_updated_at ON action_items;
DROP TRIGGER IF EXISTS trg_meeting_minutes_updated_at ON meeting_minutes;
DROP TRIGGER IF EXISTS trg_agenda_items_updated_at ON agenda_items;
DROP TRIGGER IF EXISTS trg_meeting_attendance_updated_at ON meeting_attendance;
DROP TRIGGER IF EXISTS trg_meetings_updated_at ON meetings;
DROP TRIGGER IF EXISTS trg_committee_members_updated_at ON committee_members;
DROP TRIGGER IF EXISTS trg_committees_updated_at ON committees;

DROP TABLE IF EXISTS compliance_checks;
DROP TABLE IF EXISTS action_items;
DROP TABLE IF EXISTS meeting_minutes;
DROP TABLE IF EXISTS agenda_items;
DROP TABLE IF EXISTS meeting_attendance;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS committee_members;
DROP TABLE IF EXISTS committees;

DROP FUNCTION IF EXISTS update_updated_at_column();
