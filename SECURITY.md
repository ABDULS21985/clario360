# Security Policy

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

If you discover a security vulnerability in Clario 360, please report it responsibly.

### Contact

- **Email:** security@clario360.sa
- **PGP Key:** Available upon request
- **Response time:** We acknowledge reports within 24 hours and provide a detailed response within 72 hours.

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected component(s) and version(s)
- Potential impact assessment
- Any suggested remediation (optional)

### What to Expect

1. **Acknowledgement** — Within 24 hours of your report
2. **Assessment** — We triage and assign severity within 72 hours
3. **Remediation** — Critical vulnerabilities are patched within 7 days; high within 14 days
4. **Disclosure** — We coordinate disclosure timing with the reporter
5. **Credit** — Reporters are credited in release notes (unless anonymity is requested)

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | Yes       |

Only the latest release receives security updates. Upgrade to stay protected.

## Security Architecture

### Authentication & Authorization

- **JWT (RS256)** — Asymmetric token signing with short-lived access tokens (15 min)
- **TOTP MFA** — Time-based one-time passwords for multi-factor authentication
- **bcrypt** — Password hashing with configurable cost factor (default: 12)
- **RBAC** — Role-based access control with granular permissions and wildcard matching
- **Session management** — In-memory access tokens with httpOnly refresh cookies

### Data Protection

- **Encryption at rest** — AES-256 encryption for stored files and sensitive data
- **Encryption in transit** — TLS 1.2+ for all external communication
- **HashiCorp Vault** — Dynamic credential management and transit encryption
- **PostgreSQL RLS** — Row-Level Security enforces tenant data isolation at the database layer
- **MinIO server-side encryption** — S3-compatible storage with automatic encryption

### Audit & Compliance

- **Hash chain audit trail** — Every audit entry is hash-chained for tamper detection
- **Immutable audit logs** — Append-only audit storage with integrity verification
- **CloudEvents** — Standardized event format for cross-service audit correlation
- **AI explainability** — All AI model predictions are logged with explanations; no black-box models

### Application Security

- **Input validation** — Zod (frontend) and custom validators (backend) on all inputs
- **CSRF protection** — SameSite cookies and CSRF token validation
- **Rate limiting** — Per-tenant, per-endpoint rate limiting at the API gateway
- **Content Security Policy** — Strict CSP headers on all responses
- **Dependency scanning** — Automated vulnerability scanning via Trivy, gosec, and npm audit

### Infrastructure Security

- **Kubernetes network policies** — Pod-to-pod communication restricted by namespace
- **Secret management** — No secrets in code or environment files; all secrets via Vault
- **Container scanning** — Base images scanned for CVEs before deployment
- **Air-gap support** — Full deployment capability in disconnected environments

## Security Testing

```bash
# Run security-focused tests
make test-security

# Backend static analysis
cd backend && gosec ./...

# Frontend dependency audit
cd frontend && npm audit

# Container image scanning
trivy image clario360/api-gateway:latest
```

## Responsible Disclosure

We follow a coordinated disclosure process:

1. Reporter contacts security@clario360.sa
2. We confirm receipt and begin investigation
3. We develop and test a fix
4. We release the fix and notify affected users
5. After 90 days (or upon mutual agreement), details may be published

We ask that reporters:

- Allow reasonable time for remediation before public disclosure
- Avoid accessing or modifying data belonging to other users/tenants
- Not perform denial-of-service attacks
- Not use automated scanning tools against production systems without authorization

## Security Updates

Security advisories are published via:

- GitHub Security Advisories (private repository)
- Direct notification to deployed customers
- Release notes in CHANGELOG.md
