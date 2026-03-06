package types

import "time"

// AuditAction represents the type of action being audited.
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionRead   AuditAction = "READ"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionLogin  AuditAction = "LOGIN"
	AuditActionLogout AuditAction = "LOGOUT"
	AuditActionExport AuditAction = "EXPORT"
	AuditActionImport AuditAction = "IMPORT"
)

// AuditEntry represents a single audit log entry, used across all services.
type AuditEntry struct {
	ID           ID          `json:"id" db:"id"`
	TenantID     ID          `json:"tenant_id" db:"tenant_id"`
	UserID       ID          `json:"user_id" db:"user_id"`
	Action       AuditAction `json:"action" db:"action"`
	ResourceType string      `json:"resource_type" db:"resource_type"`
	ResourceID   ID          `json:"resource_id" db:"resource_id"`
	Description  string      `json:"description" db:"description"`
	OldValue     JSONMap     `json:"old_value,omitempty" db:"old_value"`
	NewValue     JSONMap     `json:"new_value,omitempty" db:"new_value"`
	IPAddress    string      `json:"ip_address" db:"ip_address"`
	UserAgent    string      `json:"user_agent" db:"user_agent"`
	RequestID    string      `json:"request_id" db:"request_id"`
	ServiceName  string      `json:"service_name" db:"service_name"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
}
