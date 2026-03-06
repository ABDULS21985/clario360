package model

import "time"

type Role struct {
	ID           string
	TenantID     string
	Name         string
	Slug         string
	Description  string
	IsSystemRole bool
	Permissions  []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SystemRoles defines the default roles seeded on tenant creation.
var SystemRoles = []Role{
	{Name: "Super Admin", Slug: "super-admin", IsSystemRole: true, Permissions: []string{"*"}},
	{Name: "Tenant Admin", Slug: "tenant-admin", IsSystemRole: true, Permissions: []string{"tenant:*", "users:*", "roles:*", "apikeys:*"}},
	{Name: "Security Analyst", Slug: "security-analyst", IsSystemRole: true, Permissions: []string{"cyber:read", "cyber:write", "alerts:*", "remediation:read"}},
	{Name: "Security Manager", Slug: "security-manager", IsSystemRole: true, Permissions: []string{"cyber:*", "remediation:*", "alerts:*"}},
	{Name: "CISO", Slug: "ciso", IsSystemRole: true, Permissions: []string{"cyber:*", "remediation:*", "alerts:*", "reports:read", "visus:read"}},
	{Name: "Data Engineer", Slug: "data-engineer", IsSystemRole: true, Permissions: []string{"data:*", "pipelines:*", "quality:read", "lineage:read"}},
	{Name: "Data Steward", Slug: "data-steward", IsSystemRole: true, Permissions: []string{"data:read", "quality:*", "lineage:*", "data:write"}},
	{Name: "Data Analyst", Slug: "data-analyst", IsSystemRole: true, Permissions: []string{"data:read", "quality:read", "lineage:read", "reports:read"}},
	{Name: "Compliance Officer", Slug: "compliance-officer", IsSystemRole: true, Permissions: []string{"*:read", "quality:*", "lineage:read"}},
	{Name: "Legal Counsel", Slug: "legal-counsel", IsSystemRole: true, Permissions: []string{"lex:*"}},
	{Name: "Contract Manager", Slug: "contract-manager", IsSystemRole: true, Permissions: []string{"lex:read", "lex:write"}},
	{Name: "Board Secretary", Slug: "board-secretary", IsSystemRole: true, Permissions: []string{"acta:*"}},
	{Name: "Executive", Slug: "executive", IsSystemRole: true, Permissions: []string{"visus:*", "reports:read", "acta:read"}},
	{Name: "Auditor", Slug: "auditor", IsSystemRole: true, Permissions: []string{"*:read"}},
	{Name: "Viewer", Slug: "viewer", IsSystemRole: true, Permissions: []string{"*:read"}},
}
