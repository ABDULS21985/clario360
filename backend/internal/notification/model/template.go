package model

import "time"

// TemplateConfig represents a stored notification template.
type TemplateConfig struct {
	ID          string    `json:"id" db:"id"`
	TenantID    *string   `json:"tenant_id,omitempty" db:"tenant_id"`
	Channel     string    `json:"channel" db:"channel"`
	SubjectTmpl string    `json:"subject_tmpl" db:"subject_tmpl"`
	BodyTmpl    string    `json:"body_tmpl" db:"body_tmpl"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
