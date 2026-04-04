# TS-006: Full-Text Search Problems

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | TS-006                                                                |
| **Title**          | Full-Text Search Not Returning Expected Results                       |
| **Severity**       | P2 -- High                                                            |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | data-service, cyber-service, acta-service, lex-service, visus-service |
| **Namespace**      | clario360                                                             |
| **Escalation**     | Platform Engineering Lead -> Database Team Lead -> VP Engineering     |
| **SLA**            | Acknowledge within 15 minutes, resolve within 2 hours                 |

---

## Summary

This runbook addresses issues where full-text search queries return no results, incomplete results, or incorrect results across Clario360 platform services. The platform uses PostgreSQL full-text search capabilities including `tsvector` columns, `ts_query` parsing, GIN indexes, and `pg_trgm` trigram indexes for fuzzy matching. Problems can arise from stale indexes, missing `ANALYZE` runs, encoding issues in indexed data, malformed search queries, or misconfigured text search configurations.

---

## Symptoms

- Users report search returning zero results when matching records exist in the database.
- Search returns partial results, missing recently created or updated records.
- Fuzzy/similarity search (`LIKE`, `ILIKE`, trigram) returns no results despite near-matches existing.
- Search queries with special characters (accented letters, CJK characters, symbols) return empty results.
- Application logs show `tsquery` syntax errors or encoding warnings.
- Grafana dashboards show elevated search query latency or increased error rates on search endpoints.
- Prometheus alert `search_error_rate_high` or `search_latency_p99_elevated` firing.

---

## Impact Assessment

| Affected Service  | Business Impact                                                        |
|-------------------|------------------------------------------------------------------------|
| data-service      | Users cannot search datasets, reports, or data catalog entries         |
| cyber-service     | Security analysts cannot search threat intelligence or incident logs   |
| acta-service      | Document search and compliance record lookup fails                     |
| lex-service       | Legal case search and regulatory document lookup unavailable           |
| visus-service     | Dashboard search and saved report lookup fails                         |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` namespace permissions.
- `psql` client installed locally or ability to exec into a pod with `psql`.
- Access to the PostgreSQL superuser or a role with `pg_stat_statements` and index management permissions.
- Access to Grafana dashboards for search performance metrics.

---

## Diagnosis Steps

### Step 1: Identify Which Service and Database Is Affected

Determine which service's search is failing:

```bash
for SERVICE in data-service cyber-service acta-service lex-service visus-service; do
  echo "--- $SERVICE ---"
  kubectl logs -n clario360 -l app=$SERVICE --tail=100 --timestamps | grep -i -E "search|tsquery|tsvector|pg_trgm|full.text" | tail -5
done
```

### Step 2: Verify Service Health

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s http://localhost:8080/healthz | jq .
curl -s http://localhost:8080/readyz | jq .

kill $PF_PID
```

### Step 3: Test Search Endpoint Directly

```bash
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

# Test a known search term that should return results
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/search?q=test&limit=10" | jq .

# Test with a term known to exist in the database
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/search?q=<KNOWN_TERM>&limit=10" | jq '.total, .results | length'

kill $PF_PID
```

### Step 4: Connect to PostgreSQL and Check Search Indexes

Open a psql session to the affected database. Replace `<DB_NAME>` with the relevant database (`platform_core`, `cyber_db`, `data_db`, `acta_db`, `lex_db`, or `visus_db`):

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME>
```

### Step 5: Check if pg_trgm Extension Is Installed

```sql
SELECT extname, extversion FROM pg_extension WHERE extname IN ('pg_trgm', 'unaccent', 'btree_gin', 'btree_gist');
```

If `pg_trgm` is missing:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

### Step 6: List All Full-Text Search Indexes

```sql
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE indexdef ILIKE '%gin%'
   OR indexdef ILIKE '%gist%'
   OR indexdef ILIKE '%tsvector%'
   OR indexdef ILIKE '%trgm%'
ORDER BY tablename, indexname;
```

### Step 7: Check Index Health and Size

```sql
SELECT
    schemaname,
    relname AS table_name,
    indexrelname AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan AS index_scans,
    idx_tup_read AS tuples_read,
    idx_tup_fetch AS tuples_fetched
FROM pg_stat_user_indexes
WHERE indexrelname ILIKE '%search%'
   OR indexrelname ILIKE '%trgm%'
   OR indexrelname ILIKE '%gin%'
   OR indexrelname ILIKE '%fts%'
   OR indexrelname ILIKE '%tsvector%'
ORDER BY idx_scan DESC;
```

### Step 8: Check if tsvector Columns Are Populated

Replace `<TABLE>` and `<TSVECTOR_COLUMN>` with the actual table and column names:

```sql
-- Count rows with NULL or empty tsvector
SELECT
    COUNT(*) AS total_rows,
    COUNT(<TSVECTOR_COLUMN>) AS rows_with_tsvector,
    COUNT(*) - COUNT(<TSVECTOR_COLUMN>) AS rows_missing_tsvector
FROM <TABLE>;

-- Sample populated tsvector values
SELECT id, <TSVECTOR_COLUMN>
FROM <TABLE>
WHERE <TSVECTOR_COLUMN> IS NOT NULL
LIMIT 5;

-- Sample rows missing tsvector
SELECT id, created_at, updated_at
FROM <TABLE>
WHERE <TSVECTOR_COLUMN> IS NULL
ORDER BY created_at DESC
LIMIT 10;
```

### Step 9: Test a Full-Text Search Query Directly in SQL

```sql
-- Test tsvector search
SELECT id, ts_rank(<TSVECTOR_COLUMN>, plainto_tsquery('english', 'your search term')) AS rank
FROM <TABLE>
WHERE <TSVECTOR_COLUMN> @@ plainto_tsquery('english', 'your search term')
ORDER BY rank DESC
LIMIT 10;

-- Test with to_tsquery (supports boolean operators)
SELECT id, ts_rank(<TSVECTOR_COLUMN>, to_tsquery('english', 'term1 & term2')) AS rank
FROM <TABLE>
WHERE <TSVECTOR_COLUMN> @@ to_tsquery('english', 'term1 & term2')
ORDER BY rank DESC
LIMIT 10;

-- Test trigram similarity search
SELECT id, similarity(name, 'search term') AS sim
FROM <TABLE>
WHERE similarity(name, 'search term') > 0.3
ORDER BY sim DESC
LIMIT 10;
```

### Step 10: Check Table Statistics and ANALYZE Status

```sql
SELECT
    schemaname,
    relname AS table_name,
    last_analyze,
    last_autoanalyze,
    n_live_tup AS live_rows,
    n_dead_tup AS dead_rows,
    n_mod_since_analyze AS modifications_since_analyze
FROM pg_stat_user_tables
WHERE relname IN ('<TABLE1>', '<TABLE2>')
ORDER BY n_mod_since_analyze DESC;
```

### Step 11: Check for Encoding Issues in Indexed Data

```sql
-- Find rows with non-UTF8 or unusual characters
SELECT id, encode(<TEXT_COLUMN>::bytea, 'hex') AS hex_value
FROM <TABLE>
WHERE <TEXT_COLUMN> ~ '[^\x20-\x7E]'
LIMIT 10;

-- Check for null bytes that can break tsvector generation
SELECT id, <TEXT_COLUMN>
FROM <TABLE>
WHERE <TEXT_COLUMN> LIKE '%' || chr(0) || '%'
LIMIT 10;

-- Check text search configuration
SHOW default_text_search_config;

-- Verify the text search config parses correctly
SELECT * FROM ts_debug('english', 'your search term');
```

### Step 12: Check for Bloated or Corrupted Indexes

```sql
-- Check index validity
SELECT
    c.relname AS index_name,
    i.indisvalid AS is_valid,
    i.indisready AS is_ready,
    i.indislive AS is_live
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND (c.relname ILIKE '%search%'
    OR c.relname ILIKE '%gin%'
    OR c.relname ILIKE '%trgm%'
    OR c.relname ILIKE '%fts%'
    OR c.relname ILIKE '%tsvector%');
```

### Step 13: Check PostgreSQL Logs for Search-Related Errors

```bash
kubectl logs -n clario360 $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') --tail=300 | grep -i -E "tsquery|tsvector|gin|trgm|syntax error|encoding"
```

---

## Resolution Steps

### Resolution A: Run ANALYZE on Search-Enabled Tables

If `n_mod_since_analyze` is high or `last_analyze` is stale:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "ANALYZE VERBOSE <TABLE>;"
```

Run ANALYZE on all tables in the database if multiple tables are affected:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "ANALYZE VERBOSE;"
```

### Resolution B: Rebuild Full-Text Search Indexes (REINDEX)

If indexes are invalid, bloated, or corrupted:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "REINDEX INDEX CONCURRENTLY <INDEX_NAME>;"
```

To rebuild all indexes on a specific table:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "REINDEX TABLE CONCURRENTLY <TABLE>;"
```

### Resolution C: Repopulate Stale or Missing tsvector Columns

If tsvector columns have NULL values for existing records:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> <<'SQL'
-- Update tsvector column for rows where it is NULL
-- Adjust the source columns to match your schema
UPDATE <TABLE>
SET <TSVECTOR_COLUMN> = to_tsvector('english',
    coalesce(title, '') || ' ' ||
    coalesce(description, '') || ' ' ||
    coalesce(content, '')
)
WHERE <TSVECTOR_COLUMN> IS NULL;

-- Verify the update
SELECT COUNT(*) AS remaining_null FROM <TABLE> WHERE <TSVECTOR_COLUMN> IS NULL;
SQL
```

If a trigger should be maintaining the tsvector column, verify it exists:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "
SELECT trigger_name, event_manipulation, action_statement
FROM information_schema.triggers
WHERE event_object_table = '<TABLE>'
  AND action_statement ILIKE '%tsvector%';
"
```

If the trigger is missing, recreate it:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> <<'SQL'
CREATE OR REPLACE FUNCTION <TABLE>_search_update() RETURNS trigger AS $$
BEGIN
    NEW.<TSVECTOR_COLUMN> := to_tsvector('english',
        coalesce(NEW.title, '') || ' ' ||
        coalesce(NEW.description, '') || ' ' ||
        coalesce(NEW.content, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS <TABLE>_search_trigger ON <TABLE>;
CREATE TRIGGER <TABLE>_search_trigger
    BEFORE INSERT OR UPDATE ON <TABLE>
    FOR EACH ROW
    EXECUTE FUNCTION <TABLE>_search_update();
SQL
```

### Resolution D: Fix Search Query Syntax Errors

If users are submitting queries that cause `tsquery` parse errors, the application should sanitize input. As an immediate workaround, verify the service uses `plainto_tsquery` or `websearch_to_tsquery` instead of raw `to_tsquery`:

```bash
# Check application code for unsafe tsquery usage
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') -- \
  grep -r "to_tsquery" /app/ 2>/dev/null || echo "Binary or compiled service -- check source code"
```

Test safe query parsing in psql:

```sql
-- Safe: handles arbitrary user input without syntax errors
SELECT plainto_tsquery('english', 'user input with special: chars & symbols!');
SELECT websearch_to_tsquery('english', '"exact phrase" OR other terms -excluded');

-- Unsafe: will error on malformed input
-- SELECT to_tsquery('english', 'unbalanced ( parenthesis');
```

### Resolution E: Fix Character Encoding Issues

If search fails on non-ASCII data:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> <<'SQL'
-- Check database encoding
SHOW server_encoding;
SHOW client_encoding;

-- Install unaccent extension for accent-insensitive search
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Create a custom text search configuration with unaccent
CREATE TEXT SEARCH CONFIGURATION custom_english (COPY = english);
ALTER TEXT SEARCH CONFIGURATION custom_english
    ALTER MAPPING FOR hword, hword_part, word
    WITH unaccent, english_stem;

-- Rebuild tsvector with unaccent-aware config
UPDATE <TABLE>
SET <TSVECTOR_COLUMN> = to_tsvector('custom_english',
    coalesce(title, '') || ' ' ||
    coalesce(description, '') || ' ' ||
    coalesce(content, '')
);
SQL
```

### Resolution F: Adjust Trigram Similarity Threshold

If trigram-based fuzzy search returns too few or too many results:

```bash
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> <<'SQL'
-- Check current similarity threshold
SHOW pg_trgm.similarity_threshold;

-- Lower threshold for more inclusive matching (default is 0.3)
SET pg_trgm.similarity_threshold = 0.2;

-- Test the adjusted threshold
SELECT id, name, similarity(name, 'search term') AS sim
FROM <TABLE>
WHERE name % 'search term'
ORDER BY sim DESC
LIMIT 10;

-- To persist across sessions, set in postgresql.conf or per-role:
ALTER ROLE clario360 SET pg_trgm.similarity_threshold = 0.2;
SQL
```

---

## Verification

After applying a resolution, verify search functionality:

```bash
# 1. Verify indexes are valid
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "
SELECT c.relname AS index_name, i.indisvalid
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND (c.relname ILIKE '%search%' OR c.relname ILIKE '%gin%' OR c.relname ILIKE '%trgm%' OR c.relname ILIKE '%fts%');
"

# 2. Verify tsvector columns are populated
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "
SELECT COUNT(*) AS null_tsvector_rows FROM <TABLE> WHERE <TSVECTOR_COLUMN> IS NULL;
"

# 3. Test search via SQL
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "
SELECT COUNT(*) FROM <TABLE> WHERE <TSVECTOR_COLUMN> @@ plainto_tsquery('english', '<KNOWN_TERM>');
"

# 4. Test search via API
kubectl port-forward -n clario360 svc/<SERVICE> 8080:8080 &
PF_PID=$!
sleep 2

curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/search?q=<KNOWN_TERM>&limit=10" | jq '.total'

kill $PF_PID

# 5. Check service logs for search errors
kubectl logs -n clario360 -l app=<SERVICE> --since=5m --timestamps | grep -i -c -E "tsquery|search.*error|search.*fail"

# 6. Verify table statistics are fresh
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -h postgresql.clario360.svc.cluster.local -U clario360 -d <DB_NAME> -c "
SELECT relname, last_analyze, n_mod_since_analyze FROM pg_stat_user_tables WHERE relname = '<TABLE>';
"
```

---

## Post-Incident Checklist

- [ ] Confirm search queries return expected results for known test terms.
- [ ] Confirm all GIN/GiST indexes are valid (`indisvalid = true`).
- [ ] Confirm tsvector columns are fully populated (zero NULL rows).
- [ ] Confirm `ANALYZE` has been run on all affected tables.
- [ ] Verify Prometheus search error rate alerts have cleared.
- [ ] Check Grafana search latency dashboard shows normal values.
- [ ] If tsvector trigger was missing, verify it fires on new INSERT/UPDATE operations.
- [ ] Document root cause and corrective actions.
- [ ] Consider adding a scheduled job to run `ANALYZE` on high-write tables.
- [ ] Review and update monitoring to catch tsvector population drift.

---

## Related Links

| Resource                         | Link                                                                     |
|----------------------------------|--------------------------------------------------------------------------|
| PostgreSQL Full-Text Search Docs | https://www.postgresql.org/docs/16/textsearch.html                       |
| pg_trgm Extension Docs          | https://www.postgresql.org/docs/16/pgtrgm.html                          |
| Grafana Dashboards               | https://grafana.clario360.internal/dashboards                            |
| Alertmanager                     | https://alertmanager.clario360.internal                                  |
| TS-008 Connection Pool           | [TS-008-connection-pool-exhaustion.md](./TS-008-connection-pool-exhaustion.md) |
| IR-002 Database Failure          | [../incident-response/IR-002-database-failure.md](../incident-response/IR-002-database-failure.md) |
