# Cipher360: The Unified Enterprise Security, Governance & Intelligence Platform

**A Saudi-Engineered, 360-Degree Solution for Cybersecurity, Data Intelligence, Board Governance, Legal Lifecycle Management, and Executive Visibility**

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Why Cipher360 Exists](#why-cipher360-exists)
3. [Platform Architecture](#platform-architecture)
4. [The Five Integrated Suites](#the-five-integrated-suites)
   - [Cyber Suite — Security Operations & Threat Management](#1-cyber-suite--security-operations--threat-management)
   - [Data Suite — Data Intelligence & Governance](#2-data-suite--data-intelligence--governance)
   - [Acta Suite — Board & Committee Governance](#3-acta-suite--board--committee-governance)
   - [Lex Suite — Legal & Contract Lifecycle Management](#4-lex-suite--legal--contract-lifecycle-management)
   - [Visus Suite — Executive Intelligence & Visualization](#5-visus-suite--executive-intelligence--visualization)
5. [Core Platform Services](#core-platform-services)
6. [AI & Machine Learning Capabilities](#ai--machine-learning-capabilities)
7. [Security Architecture](#security-architecture)
8. [Integration Ecosystem](#integration-ecosystem)
9. [Deployment & Infrastructure](#deployment--infrastructure)
10. [Platform Statistics at a Glance](#platform-statistics-at-a-glance)
11. [Who Is Cipher360 For?](#who-is-cipher360-for)
12. [Conclusion](#conclusion)

---

## Executive Summary

In an era where organizations face converging threats across cybersecurity, data governance, regulatory compliance, and corporate governance, fragmented point solutions create dangerous blind spots. **Cipher360** eliminates these blind spots by unifying five critical enterprise domains into a single, API-first, event-driven platform — delivering a true 360-degree view of organizational risk, security posture, and governance health.

Built from the ground up as a **Kubernetes-native, multi-tenant microservices platform**, Cipher360 is engineered for the demands of modern enterprises, regulated industries, critical infrastructure, and government agencies. With 11 production services, 8 dedicated databases, 75+ data tables, 250+ API endpoints, and 70+ frontend routes, the platform delivers unmatched depth and breadth across every domain it touches.

Cipher360 is not a dashboard bolted onto third-party APIs. It is a **complete, integrated operating system for enterprise security and governance** — with its own AI engine, workflow automation, immutable audit trail, real-time event bus, and a modern frontend built with Next.js 14.

---

## Why Cipher360 Exists

Enterprise security and governance have historically been addressed through disconnected tools:

| Domain | Typical Tools | The Problem |
|--------|--------------|-------------|
| Security Operations | SIEM, SOAR, vulnerability scanners | Siloed alerting, no business context |
| Data Governance | Data catalogs, quality tools, lineage mappers | Disconnected from security posture |
| Board Governance | Meeting management, action trackers | No link to risk or compliance data |
| Legal Management | Contract management, compliance trackers | Manual processes, no automation |
| Executive Reporting | BI dashboards, spreadsheets | Stale data, no real-time visibility |

This fragmentation produces five critical failures:

1. **Context Loss** — A vulnerability in a database has no connection to the data it holds or the compliance requirements it must meet.
2. **Duplicated Effort** — Security teams, data teams, legal teams, and board secretaries maintain separate risk registers, audit trails, and compliance records.
3. **Delayed Response** — When a threat is detected, the workflow to assess impact, notify stakeholders, create remediation tasks, and update the board takes days instead of minutes.
4. **Audit Gaps** — Auditors must piece together evidence from multiple systems, each with different formats and integrity guarantees.
5. **Executive Blindness** — Leadership sees dashboards that are days old, aggregated from incompatible sources, with no drill-down capability.

**Cipher360 was built to solve all five problems simultaneously.** By placing every domain on a shared event bus, a shared identity system, a shared audit trail, and a shared workflow engine, Cipher360 ensures that an event in one domain instantly propagates context to every other domain that needs it.

---

## Platform Architecture

### Design Principles

Cipher360 is built on five architectural pillars:

- **API-First** — Every capability is exposed via RESTful APIs with OpenAPI 3.1 specifications. The frontend consumes the same APIs available to integrators.
- **Event-Driven** — Apache Kafka powers a 16-topic event bus that enables real-time cross-suite communication without tight coupling.
- **Multi-Tenant** — Row-Level Security (RLS) at the database level ensures complete tenant isolation. Every table carries a `tenant_id`, and every query is scoped.
- **Observable** — Prometheus metrics, OpenTelemetry distributed tracing, and structured logging with request ID propagation provide full operational visibility.
- **Immutable Audit** — A blockchain-style hash-chained audit trail ensures that every action is permanently recorded and tamper-evident.

### Technology Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.25 (1,584+ source files) |
| **Frontend** | Next.js 14 (App Router), TypeScript, Tailwind CSS, shadcn/ui |
| **Databases** | PostgreSQL (8 databases, partitioned tables, RLS) |
| **Message Bus** | Apache Kafka (16 topics) |
| **Cache** | Redis (rate limiting, sessions, idempotency) |
| **Object Storage** | MinIO (S3-compatible, per-tenant buckets) |
| **Metrics** | Prometheus with per-service registries |
| **Tracing** | Jaeger via OpenTelemetry |
| **Auth** | RS256 JWT, TOTP MFA, OAuth 2.0 / OIDC |

### Service Topology

```
                    ┌─────────────────────────────┐
                    │       API Gateway (:8080)     │
                    │  Rate Limiting · Circuit Breaker │
                    │  Security Headers · WebSocket    │
                    └─────────┬───────────────────────┘
                              │
          ┌───────────────────┼───────────────────────┐
          │                   │                        │
    ┌─────┴─────┐    ┌──────┴──────┐    ┌───────────┴──────────┐
    │ IAM :8081  │    │ Cyber :8085 │    │ Data Intel :8086     │
    │ Auth/RBAC  │    │ SecOps/Risk │    │ Governance/Pipelines │
    └────────────┘    └─────────────┘    └──────────────────────┘
          │                   │                        │
    ┌─────┴─────┐    ┌──────┴──────┐    ┌───────────┴──────────┐
    │Audit :8084 │    │Workflow:8083│    │ Acta :8087           │
    │Hash-Chain  │    │Engine/Tasks │    │ Board Governance     │
    └────────────┘    └─────────────┘    └──────────────────────┘
          │                   │                        │
    ┌─────┴──────┐   ┌──────┴──────┐    ┌───────────┴──────────┐
    │Notify :8090│   │ Lex :8088   │    │ Visus :8089          │
    │Multi-Chan  │   │ Legal/CLM   │    │ Exec Intelligence    │
    └────────────┘   └─────────────┘    └──────────────────────┘
          │
    ┌─────┴──────┐
    │ File :8091 │
    │ Storage/AV │
    └────────────┘
                              │
                    ┌─────────┴──────────────┐
                    │  Apache Kafka (16 topics) │
                    │  Event Bus & Dead Letter   │
                    └────────────────────────────┘
```

Every service communicates asynchronously through Kafka events, ensuring loose coupling and resilience. Synchronous communication is reserved for the API Gateway's request routing path, which includes circuit breaker protection and per-tenant rate limiting.

---

## The Five Integrated Suites

### 1. Cyber Suite — Security Operations & Threat Management

The Cyber Suite is the largest module in Cipher360, comprising 310+ Go source files and covering the full spectrum of security operations. It is not merely a SIEM or a SOAR — it is an integrated security operating system.

#### Asset Management

Every security program begins with knowing what you have. Cipher360's asset management provides:

- **Comprehensive Asset Types** — Servers, endpoints, cloud resources, IoT devices, applications, databases, and containers are all first-class entities.
- **Relationship Mapping** — Assets are connected through typed relationships: `hosts`, `runs_on`, `connects_to`, and `depends_on`. This creates a living topology of your infrastructure.
- **Bulk Operations** — Create, tag, and delete assets in batches. Import from scanners, cloud APIs, or CSV files.
- **Criticality Scoring** — Each asset receives a criticality score that feeds into the platform's risk calculations.
- **Tag-Based Organization** — Flexible tagging enables custom groupings that cut across traditional hierarchies.

#### Vulnerability Management

- **CVE Tracking** — Every vulnerability is linked to affected assets with severity assessment (CVSS scoring).
- **Vulnerability Aging** — Track how long vulnerabilities remain unpatched, with aging analytics that highlight organizational remediation velocity.
- **Per-Asset History** — View the complete vulnerability timeline for any asset.
- **Top CVE Rankings** — Identify the most prevalent and critical vulnerabilities across your environment.

#### Threat Intelligence

- **Threat Tracking** — Catalog threats by type: malware, APT groups, ransomware families, and custom categories.
- **Indicator Management** — Manage Indicators of Compromise (IoCs) including IP addresses, domains, file hashes, and URLs.
- **Bulk Indicator Import** — Ingest threat feeds and indicator lists at scale.
- **Indicator Checking** — Query any observable against your threat intelligence database for instant enrichment.

#### Alert Management

Cipher360's alert management goes far beyond a simple alert list:

- **Full Lifecycle Workflow** — Alerts progress through `New → Acknowledged → Investigating → Resolved` with complete state tracking.
- **Analyst Assignment** — Assign alerts to specific analysts with workload distribution visibility.
- **Escalation** — One-click escalation to higher tiers with context preservation.
- **Comment & Timeline** — Every alert maintains a chronological history of analyst actions, comments, and state changes.
- **Alert Correlation** — Identify related alerts and merge duplicates to reduce noise.
- **MITRE ATT&CK Mapping** — Alerts are mapped to ATT&CK tactics and techniques for strategic context.

#### Detection Rules Engine

- **Multiple Rule Types** — Sigma rules, threshold-based rules, correlation rules, and anomaly detection rules.
- **Pre-Built Templates** — A library of detection rule templates for common attack patterns.
- **Rule Testing** — Test rules against historical data before deploying to production.
- **Feedback Loop** — Mark detection accuracy to continuously improve rule quality.

#### MITRE ATT&CK Integration

- **Full Tactic & Technique Catalog** — Browse the complete ATT&CK matrix within the platform.
- **Coverage Heatmap** — Visualize your detection coverage across the ATT&CK matrix, identifying gaps.
- **Organization Mapping** — Map your specific detections, assets, and incidents to ATT&CK techniques.

#### Risk Scoring & Management

Cipher360 computes organizational risk through a multi-dimensional model:

- **Five Risk Components** — Vulnerability exposure, threat landscape, configuration posture, attack surface, and compliance status each contribute to the overall risk score.
- **Trend Analysis** — Track risk scores over time to identify improving or deteriorating areas.
- **Risk Heatmap** — Visualize risk across asset types and severity levels in a matrix view.
- **AI-Driven Recommendations** — The platform generates prioritized remediation recommendations based on risk reduction potential.

#### Remediation with Governance

Remediation in Cipher360 is not ad-hoc — it follows a governed workflow:

- **Approval Workflow** — Remediation actions (patch, configuration change, block, isolate, custom) require approval before execution.
- **Dry-Run Capability** — Test remediation actions before applying them to production systems.
- **Execution Tracking** — Monitor remediation progress from initiation through verification.
- **Rollback** — Reverse remediation actions when needed, with full audit trail.
- **Post-Verification** — Confirm that remediation was effective after execution.

#### Continuous Threat Exposure Management (CTEM)

CTEM provides a structured approach to managing your attack surface:

- **Assessment Management** — Create and manage exposure assessments with defined phases (scoping, discovery, prioritization, validation, mobilization).
- **Finding Tracking** — Catalog findings: vulnerabilities, misconfigurations, and attack paths.
- **Remediation Grouping** — Batch related findings for coordinated remediation campaigns.
- **Exposure Scoring** — Multi-factor scoring that goes beyond CVSS to include business context.
- **Report Generation** — Export assessment results for stakeholder review.

#### Data Security Posture Management (DSPM)

DSPM bridges the gap between security and data governance:

- **Data Asset Discovery** — Automatically discover data assets across your environment.
- **Sensitivity Classification** — Classify data as public, internal, confidential, or restricted.
- **Compliance Scanning** — Check data handling against regulatory requirements.
- **Policy Management** — Define and enforce data security policies with exception handling.
- **Remediation Actions** — Address data exposure issues with tracked remediation workflows.

#### User & Entity Behavior Analytics (UEBA)

- **Behavioral Profiling** — Build baseline behavior models for users and entities.
- **Activity Timeline** — Track detailed activity chronologies for any user or entity.
- **Anomaly Detection** — Alert when behavior deviates significantly from established baselines.
- **Per-User Risk Scoring** — Assign dynamic risk scores based on behavioral patterns.

#### Virtual CISO (vCISO) — AI-Powered Security Intelligence

The vCISO is Cipher360's AI advisor, providing the kind of strategic guidance typically available only from expensive consulting engagements:

- **Executive Briefings** — AI-generated security summaries (quarterly, annual, executive formats) that translate technical data into business language.
- **Risk Register** — AI-curated risk register with priority scoring and mitigation recommendations.
- **Real-Time Chat** — WebSocket-powered conversational interface for on-demand security analysis.
- **Analytical Tools:**
  - Root cause analysis
  - What-if scenario modeling
  - Natural language data filtering
  - Cross-query analysis
- **Multiple LLM Backends** — Support for OpenAI, Azure OpenAI, Anthropic Claude, local LLaMA, and BitNet models.
- **Governance Integration** — Compliance tracking, third-party risk assessment, evidence management, incident readiness evaluation.
- **Awareness Training** — Security awareness program management.
- **Budget & Maturity Planning** — Security investment planning aligned with maturity model progression.

#### Scanning & Detection Infrastructure

- **Network Scanning** — Service discovery and network mapping.
- **Cloud Asset Discovery** — Automated inventory of cloud resources.
- **Agent-Based Collection** — Endpoint data collection through lightweight agents.
- **Enrichment Pipeline** — Automatic enrichment with DNS, CVE, and geolocation data.

---

### 2. Data Suite — Data Intelligence & Governance

The Data Suite transforms raw data infrastructure into a governed, quality-assured, fully-traced data ecosystem. With 209+ Go source files, it provides enterprise-grade data governance capabilities.

#### Data Source Management

- **Multi-Database Connectors** — SQL Server, MySQL, PostgreSQL, Oracle, APIs, files, streams, and cloud storage (S3, Azure Blob, GCS).
- **Connection Testing** — Verify connectivity and credentials before saving configurations.
- **Schema Auto-Discovery** — Automatically catalog tables, columns, and data types from connected sources.
- **Metadata Browsing** — Explore schema details without leaving the platform.

#### Data Models

- **Schema-Driven Modeling** — Define data models with field-level metadata, business descriptions, and validation rules.
- **Model Versioning** — Track model evolution over time with version history.
- **Business Logic Documentation** — Attach business context to technical schemas.

#### Data Pipelines (ETL / ELT / Streaming / Batch)

- **Pipeline Definition** — Build data transformation pipelines with a visual composition interface.
- **CRON Scheduling** — Schedule pipeline execution on any CRON expression.
- **On-Demand Execution** — Trigger pipeline runs manually with full run history.
- **Detailed Execution Logs** — Trace every step of pipeline execution for debugging and auditing.
- **Transformation Builder** — Compose transformations visually with drag-and-drop logic.

#### Data Quality Management

- **Quality Rule Types** — `not_null`, `unique`, `range`, `regex`, `referential integrity`, and `custom` rules.
- **Scheduled Quality Checks** — Automate quality validation on configurable schedules.
- **Quality Scoring** — Aggregate quality metrics into scores at the dataset, source, and organizational level.
- **Failure Analysis** — Drill into quality failures to identify root causes and patterns.

#### Data Lineage

- **Graph Visualization** — Visualize data flows as directed acyclic graphs (DAGs) showing how data moves through the organization.
- **Impact Analysis** — Before making changes, understand all downstream dependencies.
- **Source Tracing** — Trace any data element back to its origin through the complete upstream chain.
- **Column-Level Lineage** — Track transformations at the individual column level.

#### Dark Data Discovery

Not all data is known and managed. Cipher360's dark data discovery finds what's been overlooked:

- **Unmodeled Table Detection** — Identify database tables that exist but aren't part of any data model.
- **Orphaned File Identification** — Find files in storage that aren't referenced by any process.
- **Stale Data Flagging** — Highlight data that hasn't been accessed or updated beyond threshold periods.
- **Classification** — Categorize discovered dark data by type and potential sensitivity.

#### Data Contradictions

- **Logical Conflict Detection** — Identify data that contradicts itself across sources (e.g., a customer marked as "active" in one system and "terminated" in another).
- **Semantic Inconsistency Detection** — Find data that uses different terminologies for the same concept.
- **Temporal Conflict Detection** — Identify time-based inconsistencies (e.g., an end date before a start date).
- **Resolution Tracking** — Track the investigation and resolution of each contradiction.

#### Analytics Engine

- **Ad-Hoc Query Execution** — Run queries against governed data sources with proper access controls.
- **Saved Query Library** — Store and organize frequently used queries.
- **Query Sharing** — Share queries with team members for collaborative analysis.

---

### 3. Acta Suite — Board & Committee Governance

The Acta Suite digitizes the entire board and committee governance lifecycle, replacing fragmented tools and manual processes with an integrated, auditable platform.

#### Committee Management

- **Committee Types** — Board, audit, risk, compensation, nomination, executive, governance, and ad hoc committees.
- **Member Roles** — Chair, vice-chair, secretary, member, and observer with role-specific permissions.
- **Quorum Rules** — Configure quorum requirements as percentage-based or fixed-count thresholds.
- **Membership Tracking** — Full history of committee composition changes.

#### Meeting Management

- **Meeting Scheduling** — Schedule meetings with support for physical, virtual, and hybrid locations.
- **Attendance Tracking** — Track attendance status: invited, confirmed, declined, present, absent, proxy, and excused.
- **Proxy Voting** — Support for proxy designation and proxy vote recording.
- **Calendar Integration** — Sync meetings with organizational calendars.

#### Agenda Management

- **Structured Agendas** — Create agenda items with presenters, time allocations, and supporting documents.
- **Item Reordering** — Drag-and-drop agenda ordering with time tracking.
- **Presenter Assignment** — Assign presenters to specific agenda items with preparation materials.

#### Voting & Resolutions

- **Voting Types** — Unanimous consent, simple majority, two-thirds majority, and roll call voting.
- **Vote Counting** — Automated vote tallying with audit trail.
- **Resolution Documentation** — Record and track formal resolutions with their voting outcomes.

#### Minutes Management

- **Workflow-Based Processing** — Minutes progress through draft → review → revision → approval → publish stages.
- **AI-Powered Generation** — Generate draft minutes automatically from meeting data, attendance, and agenda items.
- **Version Control** — Track every revision of meeting minutes with full history.
- **Publication** — Formally publish approved minutes with distribution to stakeholders.

#### Action Items

- **Cross-Meeting Tracking** — Action items persist across meetings, providing visibility into long-running commitments.
- **Assignee Management** — Assign responsibility with due dates and priority levels.
- **Status Workflow** — `Pending → In Progress → Completed → Overdue → Cancelled`.
- **Automated Monitoring** — Background processes detect overdue items and trigger notifications.

#### Governance Compliance

- **Compliance Dashboard** — Monitor adherence to governance policies across all committees.
- **Meeting Scheduling Compliance** — Track whether required meetings are being held on schedule.
- **Regulatory Adherence** — Monitor compliance with regulatory governance requirements.

---

### 4. Lex Suite — Legal & Contract Lifecycle Management

The Lex Suite provides complete contract and legal document lifecycle management, enhanced by AI-powered analysis and compliance monitoring.

#### Contract Management

- **Full Lifecycle Coverage** — Contracts progress through `Draft → Review → Negotiation → Active → Expired/Terminated` with full state tracking.
- **13 Contract Types** — Service agreements, NDAs, employment contracts, vendor agreements, licenses, leases, partnerships, consulting agreements, procurement contracts, SLAs, MOUs, amendments, and renewals.
- **Multi-Version Documents** — Track document versions with complete revision history.
- **Auto-Renewal Management** — Monitor contracts approaching renewal with configurable notice periods.
- **Risk Scoring** — Each contract receives a risk assessment score based on its terms and clauses.

#### AI-Powered Clause Extraction & Analysis

- **Automatic Clause Identification** — AI scans uploaded documents to identify and extract individual clauses.
- **Clause Classification** — Clauses are categorized: indemnification, termination, liability limitation, confidentiality, intellectual property, non-compete, payment terms, warranty, force majeure, dispute resolution, data protection, and more.
- **Per-Clause Risk Scoring** — Each clause receives an individual risk score to highlight concerning provisions.
- **Keyword Identification** — Important terms and conditions are flagged for review.

#### Contract Analysis

- **Overall Risk Assessment** — AI generates a comprehensive risk evaluation of the entire contract.
- **Missing Clause Detection** — Identify standard clauses that are absent from the contract.
- **Automatic Party Extraction** — Identify all parties to the contract from document text.
- **Date & Financial Term Extraction** — Automatically extract key dates, amounts, and financial terms.
- **Recommendations** — AI provides actionable recommendations for contract improvement.

#### Legal Document Library

- **Document Types** — Policies, regulations, templates, memos, legal opinions, filings, and correspondence.
- **Confidentiality Levels** — Public, internal, confidential, and privileged classifications.
- **Version Control** — Full version history for every document.
- **Organization & Search** — Tag-based organization with full-text search.

#### Compliance Monitoring

- **Automated Compliance Checks** — Define rules that are automatically evaluated against contracts and documents.
- **Violation Alerts** — Real-time notifications when compliance violations are detected.
- **Compliance Calendar** — View upcoming compliance deadlines and obligations.
- **Cross-Suite Synchronization** — Legal compliance events are published to the event bus for consumption by other suites.

---

### 5. Visus Suite — Executive Intelligence & Visualization

The Visus Suite provides the executive lens on the entire Cipher360 platform, transforming operational data into strategic intelligence.

#### Custom Dashboards

- **Drag-and-Drop Builder** — Create custom dashboards with a 12-column grid layout system.
- **Visibility Controls** — Dashboards can be private, team-scoped, organization-wide, or public.
- **Sharing** — Share dashboards with specific users or roles.
- **Pre-Built System Dashboards** — Out-of-the-box dashboards for common executive views.

#### Widget Library (13 Types)

| Widget | Purpose |
|--------|---------|
| KPI Card | Single metric with threshold indicators |
| Line Chart | Trend visualization over time |
| Bar Chart | Comparative analysis |
| Area Chart | Volume and trend combined |
| Pie Chart | Distribution breakdown |
| Gauge Chart | Performance against target |
| Data Table | Tabular data with sorting and filtering |
| Alert Feed | Live alert stream |
| Text Annotation | Notes and commentary |
| Sparkline | Compact trend indicators |
| Heatmap | Two-dimensional intensity mapping |
| Status Grid | Multi-item status overview |
| Trend Indicator | Directional change markers |

#### KPI Management

- **Cross-Suite KPIs** — Define KPIs that span cyber, data, governance, legal, and operational domains.
- **Threshold Configuration** — Set warning and critical thresholds with directional indicators (higher-is-better or lower-is-better).
- **Calculation Types** — Direct values, deltas, percentage changes, averages, and sums.
- **Automatic Snapshots** — KPI values are captured at configurable intervals (15 minutes to weekly) for trend analysis.

#### Executive Alerts

- **Cross-Suite Alert Aggregation** — Surface the most important alerts from all suites into a single executive view.
- **Deduplication** — Key-based deduplication with occurrence counting prevents alert fatigue.
- **Status Workflow** — `New → Viewed → Acknowledged → Actioned → Dismissed/Escalated`.
- **Escalation** — Route critical alerts to appropriate stakeholders.

#### Report Generation

- **Report Types** — Executive summaries, security posture reports, data intelligence reports, governance reports, legal reports, and custom reports.
- **Scheduled Generation** — CRON-based scheduling for automatic report creation.
- **Configurable Periods** — 7-day, 14-day, 30-day, 90-day, quarterly, annual, and custom date ranges.
- **Auto-Distribution** — Reports are automatically delivered to configured recipients via email.

---

## Core Platform Services

### API Gateway

The API Gateway is the single entry point for all client communication:

- **Dynamic Routing** — Routes requests to 11 backend services based on URL path matching.
- **Per-Tenant Rate Limiting** — Redis-backed rate limiting with separate buckets for auth, read, write, admin, upload, and WebSocket operations.
- **Circuit Breaker** — Automatically trips when backend services become unhealthy, preventing cascade failures.
- **Security Headers** — Injects CSP, HSTS, X-Frame-Options, and X-Content-Type-Options on every response.
- **WebSocket Proxy** — Transparent proxying of WebSocket connections for real-time features.
- **Request ID Propagation** — Every request receives a unique ID that is traced across all services.

### Identity & Access Management (IAM)

IAM provides the platform's authentication and authorization foundation:

- **Authentication Methods** — Email/password with bcrypt hashing, TOTP-based MFA, and OAuth 2.0/OIDC.
- **Token Management** — RS256 JWT access tokens (short-lived) with httpOnly cookie refresh tokens (long-lived).
- **Role-Based Access Control** — Wildcard-matching permission system (e.g., `cyber:alerts:*` grants all alert permissions).
- **Tenant Management** — Self-service tenant registration, team invitations, onboarding wizard, and automated provisioning.
- **Session Management** — Enumerate and revoke active sessions per user.
- **API Keys** — Generate scoped API keys for programmatic access.
- **Account Protection** — Configurable lockout thresholds after failed authentication attempts.

### Workflow Engine

The Workflow Engine powers automation across all suites:

- **Step Types:**
  - **Service Tasks** — Invoke HTTP services for automated actions.
  - **Human Tasks** — Assign form-based tasks to users with SLA monitoring.
  - **Event Tasks** — Emit Kafka events to trigger cross-suite actions.
  - **Conditions** — Branch workflow logic based on data.
  - **Timers** — Delay execution or schedule future actions.
- **Template Library** — Pre-built workflow templates for common processes.
- **Instance Recovery** — Workflow instances survive service restarts through persistent state.
- **SLA Monitoring** — Overdue human tasks are automatically flagged.

### Audit Service — Immutable Compliance Trail

The Audit Service is the platform's compliance backbone:

- **Hash-Chain Integrity** — Every audit entry includes a SHA-256 hash of its content and the hash of the previous entry, creating a blockchain-style immutable chain.
- **Database Immutability** — INSERT-only triggers at the database level prevent modification or deletion of audit records.
- **Chain Verification** — An API endpoint verifies the integrity of the entire audit chain, detecting any tampering.
- **Monthly Partitioning** — Audit logs are automatically partitioned by month for performance and retention management.
- **PII Redaction** — Sensitive data is redacted before storage to comply with privacy regulations.
- **Kafka Ingestion** — Real-time event consumption from the `platform.audit.events` topic.

### Notification Service — Multi-Channel Delivery

- **Five Delivery Channels:**
  - **In-App** — REST API with read/unread tracking and badge counts.
  - **Email** — SMTP and SendGrid with HTML template support.
  - **WebSocket** — Real-time push (up to 10 concurrent connections per user).
  - **Mobile Push** — Push notification delivery to mobile devices.
  - **Webhook** — HTTP POST to custom endpoints with HMAC signature verification.
- **User Preferences** — Per-channel, per-category notification configuration.
- **Digest Mode** — Aggregate notifications into daily or weekly digests.
- **Webhook Management** — Create custom webhooks with retry logic, exponential backoff, and dead letter queuing.
- **Idempotency** — 24-hour deduplication guard prevents duplicate notification delivery.

### File Service — Secure Document Management

- **Upload/Download** — Direct upload, presigned URL upload, streaming download, and presigned URL download.
- **Virus Scanning** — ClamAV integration scans every uploaded file, with graceful degradation if the scanner is unavailable.
- **Quarantine** — Infected files are automatically isolated in a quarantine bucket.
- **Encryption** — AES-256 encryption at rest for all stored files.
- **Versioning** — Full version history with soft delete and recovery capability.
- **Per-Tenant Isolation** — MinIO S3-compatible storage with per-tenant bucket separation.
- **Access Logging** — Complete audit trail of every file access event.

### Event Bus — Real-Time Cross-Suite Communication

Apache Kafka powers Cipher360's event-driven architecture with 16 dedicated topics:

| Topic | Events |
|-------|--------|
| `platform.iam.events` | User, role, and authentication events |
| `platform.audit.events` | Audit trail entries |
| `platform.notification.events` | Cross-suite notifications |
| `platform.workflow.events` | Workflow state changes |
| `platform.asset.events` | Asset discovery and modification |
| `platform.threat.events` | Threat intelligence updates |
| `platform.alert.events` | Alert creation and status changes |
| `platform.remediation.events` | Remediation action events |
| `platform.datasource.events` | Data source changes |
| `platform.pipeline.events` | Pipeline execution events |
| `platform.quality.events` | Data quality issues |
| `platform.contradiction.events` | Data contradiction detection |
| `platform.lineage.events` | Data lineage changes |
| `platform.acta.events` | Meeting and governance events |
| `platform.lex.events` | Contract and legal events |
| `platform.visus.events` | Dashboard and KPI events |

**Reliability features:**
- **Idempotency Guard** — Redis-backed 24-hour deduplication prevents duplicate event processing.
- **Exponential Backoff** — Failed event processing is retried with increasing delays.
- **Dead Letter Queue** — Unrecoverable failures are routed to `platform.dead-letter` for manual review, with replay capability.

---

## AI & Machine Learning Capabilities

### AI Governance Framework

Cipher360 includes a comprehensive AI governance framework (23 sub-packages) that manages the entire lifecycle of AI/ML models used within the platform:

#### Model Registry
- **Model Types** — Rule-based, statistical, ML classifiers/regressors, NLP extractors, anomaly detectors, scorers, and recommenders.
- **Risk Tiers** — Low, medium, high, and critical risk classifications.
- **Lifecycle Management** — Models progress through `Active → Deprecated → Retired` with governed transitions.

#### Version Control & Promotion
- **Version Stages** — `Development → Staging → Shadow → Production → Retired`.
- **Promotion Workflow** — Governed promotion with approval gates and audit trail.
- **Rollback** — Instant rollback to previous versions when issues are detected.

#### Shadow Mode
- **Parallel Execution** — Run shadow models alongside production models without affecting outputs.
- **Agreement Analysis** — Compare shadow and production model outputs to measure divergence.
- **Promotion Recommendations** — AI-generated recommendations: promote, keep in shadow, reject, or flag for review.

#### Drift Detection
- **Output Drift** — Population Stability Index (PSI) monitoring for distribution shifts.
- **Confidence Drift** — Track changes in model confidence distributions.
- **Volume Monitoring** — Detect unusual changes in prediction request volumes.
- **Latency Tracking** — P95 latency monitoring to catch performance degradation.
- **Accuracy Tracking** — Continuous accuracy monitoring against ground truth.

#### Explainability
- **Multiple Explanation Types** — Rule traces, feature importance scores, statistical deviation analysis, and template-based explanations.
- **Per-Version Configuration** — Each model version can have its own explainability template.
- **Prediction Logging** — Every prediction is logged with input, output, confidence, latency, and explanation for full auditability.

---

## Security Architecture

Cipher360 implements defense-in-depth across every layer:

### Authentication & Identity
- **RS256 JWT** — Industry-standard asymmetric token signing with short-lived access tokens.
- **TOTP MFA** — Time-based one-time passwords for second-factor authentication.
- **OAuth 2.0 / OIDC** — Full OpenID Connect provider with discovery, JWKS, authorization, and token endpoints.
- **Bcrypt Hashing** — Configurable-cost password hashing.
- **Account Lockout** — Automatic lockout after configurable failed authentication attempts.

### Authorization
- **Wildcard RBAC** — Permission strings support wildcards (e.g., `cyber:alerts:*`), enabling flexible role definitions without enumerating every permission.
- **Row-Level Security** — PostgreSQL RLS policies enforce tenant isolation at the database query level — not just the application layer.
- **API Key Scoping** — API keys carry their own permission sets, independent of user roles.

### Network Security
- **CORS** — Configurable Cross-Origin Resource Sharing with strict defaults.
- **Security Headers** — CSP, HSTS, X-Frame-Options, X-Content-Type-Options on every response.
- **Rate Limiting** — Per-tenant, per-endpoint Redis-backed rate limiting with six bucket categories.
- **Circuit Breaker** — Automatic protection against cascade failures.

### Data Protection
- **AES-256 at Rest** — All stored files are encrypted with AES-256.
- **TLS/SSL in Transit** — All communication is encrypted in transit.
- **HashiCorp Vault** — Integration for encryption key management.
- **PII Redaction** — Automatic redaction of personally identifiable information in audit logs.

### File Security
- **ClamAV Scanning** — Every uploaded file is scanned for malware.
- **Quarantine** — Infected files are isolated automatically.
- **Presigned URLs** — Time-limited download URLs prevent unauthorized access.
- **Per-Tenant Buckets** — Storage isolation at the tenant level.

### Audit & Compliance
- **Immutable Audit Trail** — Hash-chained, INSERT-only audit records.
- **Chain Verification** — Tamper detection through cryptographic chain validation.
- **Session Management** — Full session enumeration and revocation capability.
- **CSRF Protection** — Token-based CSRF protection for browser clients.

---

## Integration Ecosystem

Cipher360 connects to the tools your organization already uses:

### Native Integrations

| Platform | Capabilities |
|----------|-------------|
| **Slack** | OAuth connection, event subscription, slash commands, interactive messages, bidirectional notification delivery |
| **Microsoft Teams** | Message delivery and notification routing |
| **Jira** | OAuth connection, ticket creation from alerts/findings, webhook event ingestion |
| **ServiceNow** | Incident creation from security events, webhook synchronization |
| **JupyterHub** | OAuth token exchange for notebook integration, data science workflow support |

### Custom Webhooks
- **HMAC-Signed Payloads** — Every webhook delivery includes an HMAC signature for authenticity verification.
- **Retry with Backoff** — Failed deliveries are retried with exponential backoff.
- **Dead Letter Queue** — Undeliverable webhooks are queued for manual retry.
- **Webhook Testing** — Send test payloads to verify endpoint configuration.

### API-First Design
- **OpenAPI 3.1 Specification** — Complete API documentation for all 250+ endpoints.
- **API Key Authentication** — Scoped API keys for programmatic access.
- **Rate-Limited Access** — Fair-use rate limiting protects the platform while enabling integration.

---

## Deployment & Infrastructure

### Deployment Options

| Environment | Method | Notes |
|------------|--------|-------|
| **Local Development** | Docker Compose | 5 services: PostgreSQL, Redis, Kafka, MinIO, Prometheus |
| **Kubernetes** | Helm Charts | Full manifests for all 11 services with horizontal scaling |
| **Cloud** | Terraform | Infrastructure-as-code for AWS, Azure, and GCP |
| **Air-Gapped** | Bundle Deployment | Offline-capable deployment for secure environments |
| **Development** | PM2 | Process management for local multi-service development |

### CI/CD Pipeline (GitHub Actions)

- **Backend** — Lint → Test → Build → Security Scan
- **Frontend** — Lint → Test → Build → Type Check
- **Docker** — Multi-stage builds with registry push
- **Helm** — Lint and template validation

### Observability Stack

- **Prometheus** — Per-service metric registries with 14+ metrics per service.
- **Grafana** — Pre-configured dashboards for operational monitoring.
- **Jaeger** — Distributed tracing across all services via OpenTelemetry.
- **Structured Logging** — Zerolog-based structured logging with request ID propagation and sensitive data redaction.

### Health Monitoring

Every service exposes standardized health endpoints:
- `/healthz` — Liveness probe for container orchestrators.
- `/readyz` — Readiness probe including database and dependency checks.

---

## Platform Statistics at a Glance

| Category | Metric |
|----------|--------|
| **Production Services** | 11 |
| **Go Source Files** | 1,584+ |
| **Internal Packages** | 23+ |
| **Databases** | 8 (PostgreSQL) |
| **Database Tables** | 75+ |
| **SQL Migrations** | 70+ |
| **API Endpoints** | 250+ |
| **Kafka Topics** | 16 |
| **Frontend Routes** | 70+ |
| **UI Components** | ~136 |
| **Custom React Hooks** | 23 |
| **Zustand Stores** | 5 |
| **Utility Libraries** | 36+ |
| **Makefile Targets** | 30+ |
| **CI/CD Workflows** | 4+ |

---

## Who Is Cipher360 For?

### Enterprise Security Teams
Replace your SIEM, SOAR, vulnerability scanner, threat intelligence platform, and GRC tool with a single, integrated solution. Stop context-switching between tools and start seeing the full picture.

### Chief Information Security Officers (CISOs)
Get the executive visibility you need without waiting for quarterly reports. Cipher360's Visus Suite and vCISO provide real-time, AI-powered intelligence that helps you communicate risk in business terms.

### Data Governance Teams
Move beyond spreadsheet-based data catalogs. Cipher360's Data Suite provides automated discovery, quality monitoring, lineage tracking, and dark data detection — all connected to your security posture.

### Board Secretaries & Governance Officers
Digitize board governance with Acta. From meeting scheduling through AI-generated minutes to compliance tracking, every step is auditable and automated.

### Legal & Compliance Teams
Manage contracts, documents, and compliance obligations with AI-powered analysis. Cipher360's Lex Suite identifies risks, extracts clauses, and monitors compliance automatically.

### Regulated Industries
Financial services, healthcare, energy, government, and critical infrastructure organizations benefit from Cipher360's multi-tenant architecture, immutable audit trail, and comprehensive compliance capabilities.

### Managed Security Service Providers (MSSPs)
Cipher360's multi-tenant architecture is purpose-built for service providers managing multiple client environments from a single platform deployment.

---

## Conclusion

**Cipher360** represents a paradigm shift in enterprise security and governance. By unifying cybersecurity operations, data intelligence, board governance, legal management, and executive reporting into a single, event-driven platform, it eliminates the fragmentation that has plagued organizations for decades.

The platform's architecture — built on Go microservices, PostgreSQL with row-level security, Apache Kafka event streaming, and a modern Next.js frontend — ensures that it can scale from mid-size enterprises to the largest organizations in the world. Its AI governance framework, immutable audit trail, and multi-tenant design make it suitable for the most demanding regulatory environments.

With 11 production services working in concert, Cipher360 doesn't just give you a dashboard — it gives you an **operating system for enterprise risk, security, and governance**. Every alert, every data quality issue, every contract risk, every board resolution, and every executive metric flows through a single, auditable, intelligent platform.

**One platform. Complete visibility. Zero blind spots.**

---

*Cipher360 — Engineered for the enterprises that cannot afford to see anything less than the full picture.*
