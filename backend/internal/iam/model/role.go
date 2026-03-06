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
	{Name: "Tenant Admin", Slug: "tenant-admin", IsSystemRole: true, Permissions: []string{"tenant:*", "users:*", "roles:*"}},
	{Name: "Security Analyst", Slug: "security-analyst", IsSystemRole: true, Permissions: []string{"cyber:read", "cyber:write", "alerts:*"}},
	{Name: "Security Manager", Slug: "security-manager", IsSystemRole: true, Permissions: []string{"cyber:*", "remediation:approve"}},
	{Name: "Data Engineer", Slug: "data-engineer", IsSystemRole: true, Permissions: []string{"data:*", "pipelines:*"}},
	{Name: "Data Steward", Slug: "data-steward", IsSystemRole: true, Permissions: []string{"data:read", "quality:*", "lineage:*"}},
	{Name: "Legal Analyst", Slug: "legal-analyst", IsSystemRole: true, Permissions: []string{"lex:read", "lex:write"}},
	{Name: "Board Secretary", Slug: "board-secretary", IsSystemRole: true, Permissions: []string{"acta:*"}},
	{Name: "Executive", Slug: "executive", IsSystemRole: true, Permissions: []string{"visus:*", "reports:read"}},
	{Name: "Viewer", Slug: "viewer", IsSystemRole: true, Permissions: []string{"*:read"}},
}
