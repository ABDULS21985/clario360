# Clario 360 — Infrastructure as Code

Terraform infrastructure for the Clario 360 platform on Google Cloud Platform.

## Architecture

```
modules/
  networking/    VPC, subnets (public/private/isolated), Cloud NAT, firewall
  kubernetes/    GKE private cluster, 3 node pools, Helm addons (nginx, cert-manager, ArgoCD)
  database/      Cloud SQL PostgreSQL 16, 7 databases, per-service users
  redis/         Memorystore Redis (HA in production)
  kafka/         Strimzi operator, KRaft mode, 22 topics + DLQs
  storage/       GCS buckets + MinIO (S3-compatible)
  dns/           Cloud DNS with DNSSEC
  monitoring/    kube-prometheus-stack, Grafana, Loki, alerting
  security/      IAM, KMS, Binary Authorization, Network Policies
  vault/         HashiCorp Vault with auto-unseal, PKI, per-service policies
```

## Environments

| Environment | DB Tier | K8s Nodes | Kafka | Redis |
|-------------|---------|-----------|-------|-------|
| dev | db-custom-2-4096 | 1-3 workload | 1 replica | 1GB Basic |
| staging | db-custom-4-8192 | 2-6 workload | 3 replicas | 2GB Basic |
| production | db-custom-8-32768 | 3-12 workload | 3 replicas | 5GB HA |

## Quick Start

```bash
# 1. Initialize state backend
./scripts/init-backend.sh <project-id>

# 2. Plan
./scripts/plan.sh dev

# 3. Apply
./scripts/apply.sh dev
```

## Prerequisites

- Terraform >= 1.7
- gcloud CLI authenticated
- GCP project with billing enabled
- Required APIs (enabled by init-backend.sh)

## Security

- All databases use private networking (no public IP)
- KMS envelope encryption for GKE secrets and application data
- Binary Authorization enforced in production
- Network Policies (deny-by-default) in all environments
- Per-service database users with least privilege
- Vault for secrets management with Kubernetes auth
- IAP SSH tunneling only in dev/staging
