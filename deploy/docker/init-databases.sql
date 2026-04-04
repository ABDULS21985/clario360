-- =============================================================================
-- Clario 360 — Database Initialization Script
-- Creates all service databases for the database-per-service pattern
-- Run automatically by docker-compose on first PostgreSQL start
-- =============================================================================

-- Create service databases
CREATE DATABASE platform_core;
CREATE DATABASE cyber_db;
CREATE DATABASE data_db;
CREATE DATABASE acta_db;
CREATE DATABASE lex_db;
CREATE DATABASE visus_db;
CREATE DATABASE audit_db;
CREATE DATABASE notification_db;
CREATE DATABASE keycloak;

-- Grant full privileges to the clario user on each database
GRANT ALL PRIVILEGES ON DATABASE platform_core TO clario;
GRANT ALL PRIVILEGES ON DATABASE cyber_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE data_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE acta_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE lex_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE visus_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE audit_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE notification_db TO clario;
GRANT ALL PRIVILEGES ON DATABASE keycloak TO clario;

-- Enable required extensions on each database
\c platform_core;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c cyber_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c data_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c acta_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c lex_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c visus_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c audit_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c notification_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
