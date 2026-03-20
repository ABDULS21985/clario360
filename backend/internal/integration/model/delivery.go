package model

import (
	"encoding/json"
	"time"
)

type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

type DeliveryRecord struct {
	ID            string         `json:"id" db:"id"`
	TenantID      string         `json:"tenant_id" db:"tenant_id"`
	IntegrationID string         `json:"integration_id" db:"integration_id"`
	EventType     string         `json:"event_type" db:"event_type"`
	EventID       string         `json:"event_id" db:"event_id"`
	EventData     json.RawMessage `json:"event_data,omitempty" db:"event_data"`
	Status        DeliveryStatus `json:"status" db:"status"`
	Attempts      int            `json:"attempts" db:"attempts"`
	MaxAttempts   int            `json:"max_attempts" db:"max_attempts"`
	ResponseCode  *int           `json:"response_code,omitempty" db:"response_code"`
	ResponseBody  *string        `json:"response_body,omitempty" db:"response_body"`
	LastError     *string        `json:"last_error,omitempty" db:"last_error"`
	ErrorCategory *string        `json:"error_category,omitempty" db:"error_category"`
	NextRetryAt   *time.Time     `json:"next_retry_at,omitempty" db:"next_retry_at"`
	LatencyMS     *int           `json:"latency_ms,omitempty" db:"latency_ms"`
	DeliveredAt   *time.Time     `json:"delivered_at,omitempty" db:"delivered_at"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delays      []time.Duration `json:"-"`
}
