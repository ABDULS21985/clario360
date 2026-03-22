-- Migration 000026 DOWN: Revert vciso_integrations data fixes
-- Restores the original (invalid) seed values for rollback purposes.

UPDATE vciso_integrations
    SET type = 'edr'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000002';

UPDATE vciso_integrations
    SET type = 'itsm'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000003';

UPDATE vciso_integrations
    SET type = 'vulnerability_scanner'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000004';

UPDATE vciso_integrations
    SET sync_frequency = 'realtime'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000001';

UPDATE vciso_integrations
    SET sync_frequency = 'every_5min'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000002';

UPDATE vciso_integrations
    SET sync_frequency = 'hourly'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000003';

UPDATE vciso_integrations
    SET status = 'active'
    WHERE id IN (
        'aaaaaaaa-0016-1501-0000-000000000001',
        'aaaaaaaa-0016-1501-0000-000000000002',
        'aaaaaaaa-0016-1501-0000-000000000003'
    );

UPDATE vciso_integrations
    SET status = 'inactive'
    WHERE id = 'aaaaaaaa-0016-1501-0000-000000000004';
