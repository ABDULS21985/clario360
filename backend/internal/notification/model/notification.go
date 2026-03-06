package model

import (
	"encoding/json"
	"time"
)

// NotificationType identifies the kind of notification.
type NotificationType string

const (
	NotifAlertCreated         NotificationType = "alert.created"
	NotifAlertEscalated       NotificationType = "alert.escalated"
	NotifRemediationApproval  NotificationType = "remediation.approval_required"
	NotifRemediationCompleted NotificationType = "remediation.completed"
	NotifRemediationFailed    NotificationType = "remediation.failed"
	NotifTaskAssigned         NotificationType = "task.assigned"
	NotifTaskOverdue          NotificationType = "task.overdue"
	NotifTaskEscalated        NotificationType = "task.escalated"
	NotifPipelineFailed       NotificationType = "pipeline.failed"
	NotifPipelineCompleted    NotificationType = "pipeline.completed"
	NotifQualityIssue         NotificationType = "data_quality.issue_detected"
	NotifContradictionFound   NotificationType = "contradiction.detected"
	NotifContractExpiring     NotificationType = "contract.expiring"
	NotifMeetingScheduled     NotificationType = "meeting.scheduled"
	NotifMeetingReminder      NotificationType = "meeting.reminder"
	NotifKPIThreshold         NotificationType = "kpi.threshold_breached"
	NotifSystemMaintenance    NotificationType = "system.maintenance"
	NotifSecurityIncident     NotificationType = "security.incident"
	NotifPasswordExpiring     NotificationType = "password.expiring"
	NotifLoginAnomaly         NotificationType = "login.anomaly"
)

// Category values for notifications.
const (
	CategorySecurity   = "security"
	CategoryData       = "data"
	CategoryGovernance = "governance"
	CategoryLegal      = "legal"
	CategorySystem     = "system"
	CategoryWorkflow   = "workflow"
)

// Priority values for notifications.
const (
	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityLow      = "low"
)

// Delivery channel names.
const (
	ChannelInApp     = "in_app"
	ChannelEmail     = "email"
	ChannelWebSocket = "websocket"
	ChannelWebhook   = "webhook"
)

// Delivery status values.
const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
	DeliverySkipped   = "skipped"
)

// Notification represents a single user notification.
type Notification struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	UserID        string                 `json:"user_id" db:"user_id"`
	Type          NotificationType       `json:"type" db:"type"`
	Category      string                 `json:"category" db:"category"`
	Priority      string                 `json:"priority" db:"priority"`
	Title         string                 `json:"title" db:"title"`
	Body          string                 `json:"body" db:"body"`
	Data          json.RawMessage        `json:"data" db:"data"`
	ActionURL     string                 `json:"action_url" db:"action_url"`
	SourceEventID *string                `json:"source_event_id,omitempty" db:"source_event_id"`
	ReadAt        *time.Time             `json:"read_at,omitempty" db:"read_at"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// DeliveryRecord tracks each channel delivery attempt.
type DeliveryRecord struct {
	ID             string          `json:"id" db:"id"`
	NotificationID string          `json:"notification_id" db:"notification_id"`
	Channel        string          `json:"channel" db:"channel"`
	Status         string          `json:"status" db:"status"`
	Attempt        int             `json:"attempt" db:"attempt"`
	ErrorMessage   *string         `json:"error_message,omitempty" db:"error_message"`
	Metadata       json.RawMessage `json:"metadata" db:"metadata"`
	DeliveredAt    *time.Time      `json:"delivered_at,omitempty" db:"delivered_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// Webhook represents a tenant's registered webhook endpoint.
type Webhook struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	Name       string    `json:"name" db:"name"`
	URL        string    `json:"url" db:"url"`
	Secret     *string   `json:"secret,omitempty" db:"secret"`
	EventTypes []string  `json:"event_types" db:"event_types"`
	Active     bool      `json:"active" db:"active"`
	CreatedBy  string    `json:"created_by" db:"created_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// DeliveryStats aggregates delivery metrics.
type DeliveryStats struct {
	Channel   string `json:"channel"`
	Status    string `json:"status"`
	Count     int64  `json:"count"`
}

// UnreadCount holds a user's unread notification count.
type UnreadCount struct {
	Count int64 `json:"count"`
}
