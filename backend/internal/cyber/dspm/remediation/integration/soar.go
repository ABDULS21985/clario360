package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	policymodel "github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// SOARPlaybookTrigger encapsulates all the information needed to invoke a
// SOAR (Security Orchestration, Automation and Response) playbook via
// an external webhook.
type SOARPlaybookTrigger struct {
	PlaybookID    string    `json:"playbook_id"`
	RemediationID uuid.UUID `json:"remediation_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Severity      string    `json:"severity"`
	FindingType   string    `json:"finding_type"`
	WebhookURL    string    `json:"webhook_url"`
	Payload       []byte    `json:"payload,omitempty"`
}

// soarPayload is the JSON structure sent to the SOAR webhook endpoint.
type soarPayload struct {
	PlaybookID    string    `json:"playbook_id"`
	RemediationID string    `json:"remediation_id"`
	TenantID      string    `json:"tenant_id"`
	Severity      string    `json:"severity"`
	FindingType   string    `json:"finding_type"`
	TriggeredAt   time.Time `json:"triggered_at"`
}

// soarRemediationPayload is the JSON structure built from a Remediation model.
type soarRemediationPayload struct {
	RemediationID string     `json:"remediation_id"`
	TenantID      string     `json:"tenant_id"`
	FindingType   string     `json:"finding_type"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Severity      string     `json:"severity"`
	PlaybookID    string     `json:"playbook_id"`
	Status        string     `json:"status"`
	DataAssetID   string     `json:"data_asset_id,omitempty"`
	DataAssetName string     `json:"data_asset_name,omitempty"`
	CurrentStep   int        `json:"current_step"`
	TotalSteps    int        `json:"total_steps"`
	SLABreached   bool       `json:"sla_breached"`
	SLADueAt      *time.Time `json:"sla_due_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	GeneratedAt   time.Time  `json:"generated_at"`
}

// SOARConnector prepares SOAR playbook triggers and payloads for external
// integration. The actual HTTP webhook invocation is delegated to the
// caller or an integration service to maintain separation of concerns
// and allow for retry/circuit-breaker patterns.
type SOARConnector struct {
	logger zerolog.Logger
}

// NewSOARConnector constructs a SOARConnector.
func NewSOARConnector(logger zerolog.Logger) *SOARConnector {
	return &SOARConnector{
		logger: logger.With().Str("component", "soar_connector").Logger(),
	}
}

// TriggerPlaybook validates the trigger, builds the webhook payload if not
// already present, and marks it ready for dispatch. The caller is responsible
// for executing the actual HTTP POST to the webhook URL.
func (sc *SOARConnector) TriggerPlaybook(ctx context.Context, trigger *SOARPlaybookTrigger) error {
	if trigger == nil {
		return fmt.Errorf("soar connector: trigger is nil")
	}

	if trigger.PlaybookID == "" {
		return fmt.Errorf("soar connector: playbook_id is required")
	}

	if trigger.WebhookURL == "" {
		return fmt.Errorf("soar connector: webhook_url is required")
	}

	if trigger.RemediationID == uuid.Nil {
		return fmt.Errorf("soar connector: remediation_id is required")
	}

	// Build the payload if the caller did not supply one.
	if len(trigger.Payload) == 0 {
		payload := soarPayload{
			PlaybookID:    trigger.PlaybookID,
			RemediationID: trigger.RemediationID.String(),
			TenantID:      trigger.TenantID.String(),
			Severity:      trigger.Severity,
			FindingType:   trigger.FindingType,
			TriggeredAt:   time.Now().UTC(),
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("soar connector: marshal payload: %w", err)
		}

		trigger.Payload = data
	}

	sc.logger.Info().
		Str("tenant_id", trigger.TenantID.String()).
		Str("playbook_id", trigger.PlaybookID).
		Str("remediation_id", trigger.RemediationID.String()).
		Str("severity", trigger.Severity).
		Str("webhook_url", trigger.WebhookURL).
		Int("payload_size", len(trigger.Payload)).
		Msg("SOAR playbook trigger prepared")

	return nil
}

// BuildPayload constructs a JSON payload from a Remediation model suitable
// for sending to a SOAR webhook endpoint.
func (sc *SOARConnector) BuildPayload(remediation *policymodel.Remediation) ([]byte, error) {
	if remediation == nil {
		return nil, fmt.Errorf("soar connector: remediation is nil")
	}

	var dataAssetID string
	if remediation.DataAssetID != nil {
		dataAssetID = remediation.DataAssetID.String()
	}

	payload := soarRemediationPayload{
		RemediationID: remediation.ID.String(),
		TenantID:      remediation.TenantID.String(),
		FindingType:   string(remediation.FindingType),
		Title:         remediation.Title,
		Description:   remediation.Description,
		Severity:      remediation.Severity,
		PlaybookID:    remediation.PlaybookID,
		Status:        string(remediation.Status),
		DataAssetID:   dataAssetID,
		DataAssetName: remediation.DataAssetName,
		CurrentStep:   remediation.CurrentStep,
		TotalSteps:    remediation.TotalSteps,
		SLABreached:   remediation.SLABreached,
		SLADueAt:      remediation.SLADueAt,
		CreatedAt:     remediation.CreatedAt,
		GeneratedAt:   time.Now().UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("soar connector: marshal remediation payload: %w", err)
	}

	sc.logger.Debug().
		Str("remediation_id", remediation.ID.String()).
		Int("payload_size", len(data)).
		Msg("SOAR payload built")

	return data, nil
}
