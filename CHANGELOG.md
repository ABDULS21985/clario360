# Changelog

All notable changes to the Clario 360 platform are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0] — 2026-03-08

### Initial Release

Complete Clario 360 Enterprise AI Platform with five integrated suites, platform core services, and production-ready infrastructure.

### Platform Core

- **IAM Service** — User management, role-based access control with wildcard permissions, RS256 JWT authentication, TOTP MFA, tenant provisioning, invitation flow
- **API Gateway** — Chi-based reverse proxy with per-tenant rate limiting, circuit breaker (gobreaker), WebSocket proxy, request correlation, JWT validation, 14 Prometheus metrics
- **Audit Service** — Immutable audit logging with SHA-256 hash chain integrity, CloudEvents format, Kafka consumer for cross-service events, query and export capabilities
- **Workflow Engine** — BPMN-inspired workflow orchestration with expression evaluation, human task assignment, parallel/exclusive gateways, deadline escalation
- **Notification Service** — Multi-channel notifications (email, SMS, WebSocket, push), template rendering, preference management, delivery tracking, real-time WebSocket delivery
- **File Storage Service** — S3-compatible file management via MinIO, virus scanning (ClamAV), AES-256 encryption at rest, presigned URL generation, quota enforcement
- **Event Bus** — Kafka-based event processing with CloudEvents envelope, consumer groups, dead-letter queue handling, schema validation

### Cybersecurity Suite

- **Asset Management** — Discovery, classification, risk scoring, dependency mapping, software inventory
- **Threat Detection** — AI-powered threat classification, MITRE ATT&CK mapping, indicator enrichment, alert correlation
- **CTEM** — Continuous Threat Exposure Management with vulnerability assessment, exposure scoring, remediation prioritization
- **Remediation** — Governed remediation workflows with approval chains, rollback capability, SLA tracking
- **DSPM** — Data Security Posture Management with sensitive data discovery, access monitoring, compliance checking
- **Virtual CISO** — AI-driven security recommendations, risk trend analysis, executive briefings

### Data Intelligence Suite

- **Data Source Management** — Connector framework for databases, APIs, and file sources with health monitoring
- **Pipeline Orchestration** — ETL pipeline builder with scheduling, transformation steps, error handling
- **Quality Monitoring** — Automated data quality rules, anomaly detection, quality scoring, trend analysis
- **Contradiction Detection** — Cross-source data consistency checking with conflict resolution workflows
- **Data Lineage** — End-to-end lineage tracking with visualization, impact analysis, compliance traceability
- **Dark Data Discovery** — Unused data identification, classification, and retention recommendations
- **Analytics Dashboard** — Data health metrics, pipeline performance, quality trends

### Board Governance Suite (Acta)

- **Committee Management** — Committee creation, member assignment, term tracking, quorum rules
- **Meeting Management** — Agenda builder, scheduling, attendee management, document distribution
- **AI Minutes Generation** — Automated meeting minutes with action item extraction and decision logging
- **Action Item Tracking** — Assignment, deadlines, status tracking, escalation, completion verification
- **Governance Compliance** — Board composition requirements, attendance tracking, disclosure management

### Legal Operations Suite (Lex)

- **Contract Management** — Full contract lifecycle from draft through execution and renewal
- **AI Clause Extraction** — Automated identification and classification of contract clauses
- **Risk Scoring** — Contract risk assessment based on clause analysis and compliance requirements
- **Expiry Monitoring** — Automated notifications for upcoming contract renewals and expirations
- **Compliance Alerts** — Regulatory compliance checking against contract terms
- **Obligation Tracking** — Contractual obligation monitoring with deadline enforcement

### Executive Intelligence Suite (Visus360)

- **Cross-Suite Dashboard** — Unified view across all platform suites with real-time data
- **KPI Engine** — Configurable KPI definitions, calculations, thresholds, and trend analysis
- **Widget Framework** — Customizable dashboard widgets (charts, tables, gauges, timelines)
- **Executive Alerts** — Priority-based alerting for KPI breaches and critical events
- **Report Generation** — Automated and on-demand report creation with scheduling

### Frontend

- **Authentication System** — Login, registration, MFA setup, password reset, BFF pattern with in-memory tokens
- **Dashboard Shell** — Collapsible sidebar, command palette, notification dropdown, WebSocket connection status
- **Component Library** — Charts (bar, pie, area, gauge, line), KPI cards, data tables, form components, severity indicators
- **User Management** — Full CRUD for users, roles, permissions with permission tree and wildcard support
- **Workflow UI** — Task inbox, form builder, workflow visualization, real-time status updates
- **Notification Center** — Grouped notifications, infinite scroll, mark read/unread, real-time delivery
- **Suite Pages** — Complete UI for all five suites with search, filtering, detail panels

### AI Governance

- **Model Registry** — Version-controlled model registration with metadata and deployment tracking
- **Prediction Logging** — All AI predictions logged with inputs, outputs, and confidence scores
- **Explainability** — Rule-based, statistical, and template-based explanation generators
- **Drift Detection** — Model performance monitoring with drift alerts and retraining triggers
- **Shadow Deployment** — Side-by-side model comparison with automated evaluation

### Infrastructure

- **Terraform** — GCP infrastructure modules for networking, Kubernetes, databases, Redis, Kafka
- **Helm Charts** — Complete Kubernetes deployment manifests for all services
- **Docker** — Multi-stage Dockerfiles for backend (Go) and frontend (Next.js)
- **Monitoring** — Prometheus metrics, Grafana dashboards, Alertmanager rules, Jaeger tracing
- **CI/CD** — GitHub Actions workflows for lint, test, build, scan, and deploy
- **Air-Gap Support** — Offline bundle creation and deployment scripts
- **Backup/DR** — Automated backup scripts with disaster recovery runbook
- **Escrow** — Source code escrow packaging with integrity verification
- **Load Testing** — k6 scenarios for smoke, load, stress, and soak testing

### Compliance

- ISO 27001 control mapping (93 controls)
- NCA Essential Cybersecurity Controls mapping
- SAMA Cyber Security Framework mapping
- NIST Cybersecurity Framework mapping

### Operational Runbooks

- 35 runbooks covering deployment, incident response, operations, scaling, and troubleshooting
- Each runbook includes exact commands, expected outputs, and escalation procedures
