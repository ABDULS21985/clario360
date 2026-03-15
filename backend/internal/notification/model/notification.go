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
	NotifActionItemAssigned   NotificationType = "action_item.assigned"
	NotifActionItemOverdue    NotificationType = "action_item.overdue"
	NotifMinutesApproved      NotificationType = "minutes.approved"
	NotifKPIThreshold         NotificationType = "kpi.threshold_breached"
	NotifSystemMaintenance    NotificationType = "system.maintenance"
	NotifSecurityIncident     NotificationType = "security.incident"
	NotifPasswordExpiring     NotificationType = "password.expiring"
	NotifLoginAnomaly         NotificationType = "login.anomaly"
	NotifContractCreated      NotificationType = "contract.created"
	NotifAnalysisReady        NotificationType = "analysis.ready"
	NotifClauseRiskFlagged    NotificationType = "clause.risk_flagged"
	NotifWorkflowFailed       NotificationType = "workflow.failed"
	NotifWorkflowCompleted    NotificationType = "workflow.completed"
	NotifWelcome              NotificationType = "welcome"
	NotifMalwareDetected      NotificationType = "malware.detected"
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
	ID            string           `json:"id" db:"id"`
	TenantID      string           `json:"tenant_id" db:"tenant_id"`
	UserID        string           `json:"user_id" db:"user_id"`
	Type          NotificationType `json:"type" db:"type"`
	Category      string           `json:"category" db:"category"`
	Priority      string           `json:"priority" db:"priority"`
	Title         string           `json:"title" db:"title"`
	Body          string           `json:"body" db:"body"`
	Data          json.RawMessage  `json:"data" db:"data"`
	ActionURL     string           `json:"action_url" db:"action_url"`
	SourceEventID *string          `json:"source_event_id,omitempty" db:"source_event_id"`
	Read          bool             `json:"read" db:"-"`
	ReadAt        *time.Time       `json:"read_at,omitempty" db:"read_at"`
	CreatedAt     time.Time        `json:"created_at" db:"created_at"`
}

// ComputeRead sets the Read field based on ReadAt.
func (n *Notification) ComputeRead() {
	n.Read = n.ReadAt != nil
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

// WebhookRetryPolicy defines the retry behaviour for webhook delivery.
type WebhookRetryPolicy struct {
	MaxRetries         int    `json:"max_retries"`
	BackoffType        string `json:"backoff_type"`
	InitialDelaySeconds int   `json:"initial_delay_seconds"`
}

// DefaultRetryPolicy returns sensible defaults.
func DefaultRetryPolicy() WebhookRetryPolicy {
	return WebhookRetryPolicy{
		MaxRetries:          3,
		BackoffType:         "exponential",
		InitialDelaySeconds: 10,
	}
}

// Webhook represents a tenant's registered webhook endpoint.
type Webhook struct {
	ID              string              `json:"id" db:"id"`
	TenantID        string              `json:"-" db:"tenant_id"`
	Name            string              `json:"name" db:"name"`
	URL             string              `json:"url" db:"url"`
	Secret          *string             `json:"secret,omitempty" db:"secret"`
	Events          []string            `json:"events" db:"event_types"`
	Active          bool                `json:"-" db:"active"`
	Status          string              `json:"status" db:"-"`
	Headers         map[string]string   `json:"headers" db:"-"`
	HeadersRaw      json.RawMessage     `json:"-" db:"headers"`
	RetryPolicy     WebhookRetryPolicy  `json:"retry_policy" db:"-"`
	RetryPolicyRaw  json.RawMessage     `json:"-" db:"retry_policy"`
	LastTriggeredAt *time.Time          `json:"last_triggered_at" db:"last_triggered_at"`
	SuccessCount    int64               `json:"success_count" db:"success_count"`
	FailureCount    int64               `json:"failure_count" db:"failure_count"`
	CreatedBy       string              `json:"created_by" db:"created_by"`
	CreatedAt       time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at" db:"updated_at"`
}

// ComputeDerived sets the Status, Headers, and RetryPolicy from raw DB fields.
func (w *Webhook) ComputeDerived() {
	// Compute status from active + failure_count
	if !w.Active {
		w.Status = "inactive"
	} else if w.FailureCount > 0 && w.SuccessCount == 0 {
		w.Status = "failing"
	} else {
		w.Status = "active"
	}

	// Deserialize headers
	if len(w.HeadersRaw) > 0 {
		var h map[string]string
		if err := json.Unmarshal(w.HeadersRaw, &h); err == nil {
			w.Headers = h
		}
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}

	// Deserialize retry_policy
	if len(w.RetryPolicyRaw) > 0 {
		var rp WebhookRetryPolicy
		if err := json.Unmarshal(w.RetryPolicyRaw, &rp); err == nil {
			w.RetryPolicy = rp
		} else {
			w.RetryPolicy = DefaultRetryPolicy()
		}
	} else {
		w.RetryPolicy = DefaultRetryPolicy()
	}
}

// WebhookDelivery represents a single delivery attempt to a webhook endpoint.
type WebhookDelivery struct {
	ID             string          `json:"id" db:"id"`
	WebhookID      string          `json:"webhook_id" db:"webhook_id"`
	EventType      string          `json:"event_type" db:"event_type"`
	Status         string          `json:"status" db:"status"`
	RequestURL     string          `json:"request_url" db:"request_url"`
	RequestBody    json.RawMessage `json:"request_body" db:"request_body"`
	ResponseStatus *int            `json:"response_status" db:"response_status"`
	ResponseBody   *string         `json:"response_body" db:"response_body"`
	DurationMS     *int            `json:"duration_ms" db:"duration_ms"`
	AttemptCount   int             `json:"attempt_count" db:"attempt"`
	NextRetryAt    *time.Time      `json:"next_retry_at" db:"next_retry_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// DeliveryStats aggregates delivery metrics.
type DeliveryStats struct {
	Channel string `json:"channel"`
	Status  string `json:"status"`
	Count   int64  `json:"count"`
}

// RichDeliveryStats is the frontend-compatible delivery statistics response.
type RichDeliveryStats struct {
	Period            string                    `json:"period"`
	TotalSent         int64                     `json:"total_sent"`
	Delivered         int64                     `json:"delivered"`
	Failed            int64                     `json:"failed"`
	DeliveryRate      float64                   `json:"delivery_rate"`
	ByChannel         map[string]ChannelStats   `json:"by_channel"`
	ByType            map[string]int64          `json:"by_type"`
	ByDay             []DayStats                `json:"by_day"`
	AvgDeliveryTimeMS int64                     `json:"avg_delivery_time_ms"`
}

// ChannelStats holds per-channel delivery metrics.
type ChannelStats struct {
	Sent              int64 `json:"sent"`
	Delivered         int64 `json:"delivered"`
	Failed            int64 `json:"failed"`
	AvgDeliveryTimeMS int64 `json:"avg_delivery_time_ms"`
}

// DayStats holds daily delivery metrics.
type DayStats struct {
	Date      string `json:"date"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Failed    int64  `json:"failed"`
}

// UnreadCount holds a user's unread notification count.
type UnreadCount struct {
	Count int64 `json:"count"`
}
