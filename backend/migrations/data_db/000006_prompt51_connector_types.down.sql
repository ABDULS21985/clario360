ALTER TABLE data_sources
    DROP CONSTRAINT IF EXISTS data_sources_type_check;

ALTER TABLE data_sources
    ADD CONSTRAINT data_sources_type_check CHECK (
        type IN (
            'postgresql',
            'mysql',
            'mssql',
            'api',
            'csv',
            's3',
            'stream'
        )
    );
