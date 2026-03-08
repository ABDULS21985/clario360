package model

import (
	"encoding/json"
	"time"
)

type IntegrationType string

const (
	IntegrationTypeSlack       IntegrationType = "slack"
	IntegrationTypeTeams       IntegrationType = "teams"
	IntegrationTypeJira        IntegrationType = "jira"
	IntegrationTypeServiceNow  IntegrationType = "servicenow"
	IntegrationTypeWebhook     IntegrationType = "webhook"
)

type IntegrationStatus string

const (
	IntegrationStatusActive       IntegrationStatus = "active"
	IntegrationStatusInactive     IntegrationStatus = "inactive"
	IntegrationStatusError        IntegrationStatus = "error"
	IntegrationStatusSetupPending IntegrationStatus = "setup_pending"
)

type EventFilter struct {
	EventTypes    []string `json:"event_types,omitempty"`
	Severities    []string `json:"severities,omitempty"`
	Suites        []string `json:"suites,omitempty"`
	MinConfidence float64  `json:"min_confidence,omitempty"`
}

type Integration struct {
	ID              string            `json:"id" db:"id"`
	TenantID        string            `json:"tenant_id" db:"tenant_id"`
	Type            IntegrationType   `json:"type" db:"type"`
	Name            string            `json:"name" db:"name"`
	Description     string            `json:"description" db:"description"`
	ConfigEncrypted []byte            `json:"-" db:"config_encrypted"`
	ConfigNonce     []byte            `json:"-" db:"config_nonce"`
	ConfigKeyID     string            `json:"-" db:"config_key_id"`
	Status          IntegrationStatus `json:"status" db:"status"`
	ErrorMessage    *string           `json:"error_message,omitempty" db:"error_message"`
	ErrorCount      int               `json:"error_count" db:"error_count"`
	LastErrorAt     *time.Time        `json:"last_error_at,omitempty" db:"last_error_at"`
	EventFilters    []EventFilter     `json:"event_filters" db:"event_filters"`
	LastUsedAt      *time.Time        `json:"last_used_at,omitempty" db:"last_used_at"`
	DeliveryCount   int64             `json:"delivery_count" db:"delivery_count"`
	CreatedBy       string            `json:"created_by" db:"created_by"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time        `json:"-" db:"deleted_at"`
	SanitizedConfig map[string]any    `json:"config,omitempty"`
}

func (i *Integration) ConfigJSON() json.RawMessage {
	if len(i.ConfigEncrypted) == 0 {
		return nil
	}
	return json.RawMessage(i.ConfigEncrypted)
}

type SlackConfig struct {
	BotToken           string `json:"bot_token,omitempty"`
	ChannelID          string `json:"channel_id,omitempty"`
	TeamID             string `json:"team_id,omitempty"`
	TeamName           string `json:"team_name,omitempty"`
	SigningSecret      string `json:"signing_secret,omitempty"`
	IncomingWebhookURL string `json:"incoming_webhook_url,omitempty"`
	ThreadPerAlert     bool   `json:"thread_per_alert,omitempty"`
	IncludeExplanation bool   `json:"include_explanation,omitempty"`
}

type TeamsConfig struct {
	BotAppID       string `json:"bot_app_id,omitempty"`
	BotPassword    string `json:"bot_password,omitempty"`
	ServiceURL     string `json:"service_url,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
}

type JiraConfig struct {
	BaseURL         string            `json:"base_url,omitempty"`
	CloudID         string            `json:"cloud_id,omitempty"`
	ProjectKey      string            `json:"project_key,omitempty"`
	IssueTypeID     string            `json:"issue_type_id,omitempty"`
	AuthToken       string            `json:"auth_token,omitempty"`
	RefreshToken    string            `json:"refresh_token,omitempty"`
	WebhookSecret   string            `json:"webhook_secret,omitempty"`
	PriorityMapping map[string]string `json:"priority_mapping,omitempty"`
	StatusMapping   map[string]string `json:"status_mapping,omitempty"`
	CustomFields    map[string]any    `json:"custom_fields,omitempty"`
}

type ServiceNowConfig struct {
	InstanceURL     string            `json:"instance_url,omitempty"`
	AuthType        string            `json:"auth_type,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`
	OAuthToken      string            `json:"oauth_token,omitempty"`
	AssignmentGroup string            `json:"assignment_group,omitempty"`
	CallerID        string            `json:"caller_id,omitempty"`
	Category        string            `json:"category,omitempty"`
	Subcategory     string            `json:"subcategory,omitempty"`
	WebhookSecret   string            `json:"webhook_secret,omitempty"`
	StatusMapping   map[string]string `json:"status_mapping,omitempty"`
	CustomFields    map[string]any    `json:"custom_fields,omitempty"`
}

type WebhookConfig struct {
	URL         string            `json:"url,omitempty"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Secret      string            `json:"secret,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
}
