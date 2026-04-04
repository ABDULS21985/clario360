# Clario 360 — CTI Schema Report

Generated: 2026-04-03

## 1. Overview

Six migrations (000027–000032) introduce 15 new tables into `cyber_db` and backfill missing audit columns across the CTI schema. All tables are tenant-scoped with RLS, use UUID primary keys, and follow the existing migration conventions.

## 2. Migration Summary

| Migration | Description | Tables Created |
|-----------|-------------|---------------|
| 000027 | CTI Reference Tables | `cti_threat_severity_levels`, `cti_threat_categories`, `cti_geographic_regions`, `cti_industry_sectors`, `cti_data_sources` |
| 000028 | CTI Threat Activity | `cti_threat_events`, `cti_threat_event_tags` |
| 000029 | CTI Campaigns & Actors | `cti_threat_actors`, `cti_campaigns`, `cti_campaign_events`, `cti_campaign_iocs` |
| 000030 | CTI Brand Abuse | `cti_monitored_brands`, `cti_brand_abuse_incidents` |
| 000031 | CTI Aggregation / Dashboard | `cti_geo_threat_summary`, `cti_sector_threat_summary`, `cti_executive_snapshot` |
| 000032 | CTI Audit Backfill | Adds missing `created_at` / `updated_at` / `created_by` / `updated_by` columns where required |

## 3. Entity Relationship Diagram

```
                          ┌─────────────────────────┐
                          │  cti_threat_severity_    │
                          │  levels                  │
                          │  (5 rows)                │
                          └────────┬────────────────┘
                                   │ severity_id
         ┌─────────────────────────┼──────────────────────────┐
         │                         │                          │
         ▼                         ▼                          ▼
┌────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐
│  cti_threat_events │  │  cti_campaigns       │  │  (used by dashboards)│
│  (550 rows)        │  │  (12 rows)           │  └──────────────────────┘
│                    │  │                      │
│  ├─ severity_id ──►│  │  ├─ severity_id     │
│  ├─ category_id ──►│  │  ├─ primary_actor ──┼──► cti_threat_actors (15)
│  ├─ source_id ────►│  │  ├─ target_sectors  │     ├─ origin_region_id ──► cti_geographic_regions
│  ├─ origin_region ►│  │  ├─ target_regions  │     └─ origin_country_code
│  ├─ target_sector ►│  │  └─ mitre_tech_ids  │
│  └─ mitre_tech_ids │  └──────────┬───────────┘
└────────┬───────────┘             │
         │                         │
         │    ┌────────────────────┤
         │    │                    │
         ▼    ▼                    ▼
┌──────────────────┐    ┌──────────────────┐
│ cti_campaign_    │    │ cti_campaign_    │
│ events (388)     │    │ iocs (240)       │
│ M:N junction     │    │                  │
│ campaign ◄─► evt │    │ ├─ campaign_id   │
└──────────────────┘    │ ├─ source_id ───►│ cti_data_sources (8)
                        └──────────────────┘

┌─────────────────────┐     ┌─────────────────────────┐
│ cti_threat_         │     │ cti_threat_categories   │
│ event_tags          │     │ (15 rows)               │
│ ├─ event_id ───────►│     │ ├─ mitre_tactic_ids     │
└─────────────────────┘     └─────────────────────────┘

┌─────────────────────┐     ┌─────────────────────────┐
│ cti_geographic_     │     │ cti_industry_sectors    │
│ regions (56 rows)   │     │ (13 rows)               │
│ ├─ parent_region_id │     │ ├─ naics_code           │
│ (self-referencing)  │     └─────────────────────────┘
└─────────────────────┘

┌─────────────────────┐     ┌─────────────────────────┐
│ cti_monitored_      │     │ cti_brand_abuse_        │
│ brands (10 rows)    │◄────│ incidents (45 rows)     │
│ ├─ domain_pattern   │     │ ├─ brand_id             │
│ ├─ keywords[]       │     │ ├─ region_id ──────────►│ cti_geographic_regions
└─────────────────────┘     │ ├─ source_id ──────────►│ cti_data_sources
                            │ ├─ takedown_status      │
                            └─────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│  AGGREGATION / DASHBOARD TABLES (pre-computed)            │
├──────────────────────┬─────────────────────┬─────────────┤
│ cti_geo_threat_      │ cti_sector_threat_  │ cti_exec_   │
│ summary (48 rows)    │ summary (39 rows)   │ snapshot(1) │
│ per-country/city/    │ per-sector/period   │ KPI per     │
│ period               │                     │ tenant      │
└──────────────────────┴─────────────────────┴─────────────┘
```

## 4. Table Details

### 4.1 Reference Tables (migration 000027)

| Table | PK | Unique Constraint | Key Columns |
|-------|----|--------------------|-------------|
| `cti_threat_severity_levels` | id | (tenant_id, code) | code, label, color_hex, sort_order |
| `cti_threat_categories` | id | (tenant_id, code) | code, label, description, mitre_tactic_ids[] |
| `cti_geographic_regions` | id | (tenant_id, code) | code, label, parent_region_id (self-FK), lat/lng, iso_country_code |
| `cti_industry_sectors` | id | (tenant_id, code) | code, label, description, naics_code |
| `cti_data_sources` | id | (tenant_id, name) | name, source_type, url, reliability_score, poll_interval_seconds |

### 4.2 Threat Activity (migration 000028)

**`cti_threat_events`** — core event stream

| Column | Type | FK / Notes |
|--------|------|-----------|
| severity_id | UUID | → cti_threat_severity_levels |
| category_id | UUID | → cti_threat_categories |
| source_id | UUID | → cti_data_sources |
| origin_region_id | UUID | → cti_geographic_regions |
| target_sector_id | UUID | → cti_industry_sectors |
| origin_country_code | VARCHAR(3) | ISO country |
| ioc_type / ioc_value | VARCHAR / TEXT | IOC pair |
| mitre_technique_ids | TEXT[] | GIN indexed |
| raw_payload | JSONB | GIN indexed |
| confidence_score | DECIMAL(3,2) | 0.00–1.00 |

Indexes: 9 indexes including GIN on mitre_technique_ids and raw_payload.

**`cti_threat_event_tags`** — free-form tags, unique per (tenant, event, tag).

### 4.3 Campaigns & Actors (migration 000029)

**`cti_threat_actors`** — 15 threat actor profiles with aliases[], origin, sophistication, motivation, risk_score.

**`cti_campaigns`** — 12 campaigns with status lifecycle (active/monitoring/dormant/resolved/archived), linked to actor via primary_actor_id FK, target_sectors[] and target_regions[] as UUID arrays.

**`cti_campaign_events`** — M:N junction (campaign ↔ event), unique per (tenant, campaign, event).

**`cti_campaign_iocs`** — campaign-specific IOCs with confidence scores and active flags.

### 4.4 Brand Abuse (migration 000030)

**`cti_monitored_brands`** — 10 brands with domain_pattern and keywords[].

**`cti_brand_abuse_incidents`** — 45 incidents with abuse_type, risk_level, WHOIS/hosting data, takedown lifecycle (detected → reported → takedown_requested → taken_down).

### 4.5 Aggregation (migration 000031)

**`cti_geo_threat_summary`** — pre-computed per (country, city, period). 48 rows across 3 periods (24h, 7d, 30d).

**`cti_sector_threat_summary`** — pre-computed per (sector, period). 39 rows across 3 periods.

**`cti_executive_snapshot`** — single row per tenant with KPI metrics (unique on tenant_id).

## 5. RLS Policies

All 15 tables have 4 RLS policies each (60 policies total):

```sql
tenant_isolation  — USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
tenant_insert     — FOR INSERT WITH CHECK (...)
tenant_update     — FOR UPDATE USING (...) WITH CHECK (...)
tenant_delete     — FOR DELETE USING (...)
```

RLS is `ENABLE`d and `FORCE`d on every table.

## 6. Seed Data Summary

| Table | Records | Notes |
|-------|---------|-------|
| cti_threat_severity_levels | 5 | critical, high, medium, low, informational |
| cti_threat_categories | 15 | APT through destructive |
| cti_geographic_regions | 56 | 6 continents + 16 sub-regions + 34 countries |
| cti_industry_sectors | 13 | technology through manufacturing |
| cti_data_sources | 8 | OSINT, commercial, government, internal, dark web |
| cti_threat_actors | 15 | State-sponsored (8), cybercriminal (4), hacktivist (1), insider (1), other (1) |
| cti_campaigns | 12 | active (6), monitoring (3), dormant (2), resolved (1) |
| cti_threat_events | 550 | ~15% critical, ~25% high, ~35% medium, ~25% low |
| cti_campaign_iocs | 240 | ~20 per campaign (IPs, domains, hashes, CVEs, URLs) |
| cti_campaign_events | 388 | ~32 events per campaign average |
| cti_monitored_brands | 10 | Fictional brands |
| cti_brand_abuse_incidents | 45 | Mixed risk levels and takedown statuses |
| cti_geo_threat_summary | 48 | 16 cities × 3 periods |
| cti_sector_threat_summary | 39 | 13 sectors × 3 periods |
| cti_executive_snapshot | 1 | Single KPI row for dev tenant |
| **Total** | **1,445** | |

## 7. Seeder

**Location:** `backend/cmd/cti-seeder/main.go`

**Usage:**
```bash
GOWORK=off go run ./cmd/cti-seeder/ \
  --db-url="postgres://clario:clario_dev_pass@localhost:5432/cyber_db?sslmode=disable" \
  --tenant-id="aaaaaaaa-0000-0000-0000-000000000001"
```

**Features:**
- Idempotent (ON CONFLICT DO NOTHING / DO UPDATE)
- Sets `app.current_tenant_id` for RLS compatibility
- Deterministic seed (default 42)
- Single transaction (all or nothing)
- Batched inserts for events and IOCs

## 8. Performance

Query: 24h threat events, sorted descending, limit 50
- Execution time: **0.616ms**
- Well under 10ms target
- Indexes available for production volumes
