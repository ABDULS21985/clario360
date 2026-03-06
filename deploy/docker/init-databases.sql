-- Initialize per-service databases for local development
CREATE DATABASE IF NOT EXISTS keycloak;
-- All Clario services share the clario360 database with separate schemas
-- This keeps local dev simple while supporting schema-level isolation

\c clario360;

CREATE SCHEMA IF NOT EXISTS iam;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS cyber;
CREATE SCHEMA IF NOT EXISTS data_suite;
CREATE SCHEMA IF NOT EXISTS acta;
CREATE SCHEMA IF NOT EXISTS lex;
CREATE SCHEMA IF NOT EXISTS visus;
CREATE SCHEMA IF NOT EXISTS workflow;

-- Grant usage to the clario user
GRANT ALL PRIVILEGES ON SCHEMA iam TO clario;
GRANT ALL PRIVILEGES ON SCHEMA audit TO clario;
GRANT ALL PRIVILEGES ON SCHEMA cyber TO clario;
GRANT ALL PRIVILEGES ON SCHEMA data_suite TO clario;
GRANT ALL PRIVILEGES ON SCHEMA acta TO clario;
GRANT ALL PRIVILEGES ON SCHEMA lex TO clario;
GRANT ALL PRIVILEGES ON SCHEMA visus TO clario;
GRANT ALL PRIVILEGES ON SCHEMA workflow TO clario;
