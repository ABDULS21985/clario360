package consumer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/lex/service"
)

type LexConsumer struct {
	compliance *service.ComplianceService
	workflows  *service.WorkflowService
	consumer   *events.Consumer
	logger     zerolog.Logger
}

func NewLexConsumer(compliance *service.ComplianceService, workflows *service.WorkflowService, consumer *events.Consumer, logger zerolog.Logger) *LexConsumer {
	handler := &LexConsumer{
		compliance: compliance,
		workflows:  workflows,
		consumer:   consumer,
		logger:     logger.With().Str("component", "lex-consumer").Logger(),
	}
	if consumer != nil {
		consumer.Subscribe(events.Topics.FileEvents, handler)
		consumer.Subscribe(events.Topics.WorkflowEvents, handler)
	}
	return handler
}

func (c *LexConsumer) Start(ctx context.Context) error {
	if c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *LexConsumer) Stop() error {
	if c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *LexConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.workflow.instance.completed":
		return c.handleWorkflowCompleted(ctx, event)
	case "com.clario360.file.scan.infected",
		"com.clario360.file.quarantined",
		"com.clario360.file.scan.error",
		"com.clario360.file.expired":
		return c.handleFileIntegrityEvent(ctx, event)
	default:
		return nil
	}
}

func (c *LexConsumer) handleWorkflowCompleted(ctx context.Context, event *events.Event) error {
	if c.workflows == nil {
		return nil
	}
	var payload struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.InstanceID) == "" {
		return nil
	}
	workflowInstanceID, err := uuid.Parse(payload.InstanceID)
	if err != nil {
		return nil
	}
	return c.workflows.AdvanceOnWorkflowCompletion(ctx, workflowInstanceID)
}

func (c *LexConsumer) handleFileIntegrityEvent(ctx context.Context, event *events.Event) error {
	if c.compliance == nil {
		return nil
	}

	tenantID, err := uuid.Parse(strings.TrimSpace(event.TenantID))
	if err != nil {
		return nil
	}

	var payload struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}
	fileID, err := uuid.Parse(strings.TrimSpace(payload.FileID))
	if err != nil {
		return nil
	}

	description := fileIntegrityDescription(event.Type)
	return c.compliance.HandleFileIntegrityEvent(ctx, tenantID, fileID, event.Type, description)
}

func fileIntegrityDescription(eventType string) string {
	switch eventType {
	case "com.clario360.file.scan.infected":
		return "The linked file was identified as infected during malware scanning."
	case "com.clario360.file.quarantined":
		return "The linked file was quarantined by the file service."
	case "com.clario360.file.scan.error":
		return "The linked file could not be fully verified because the malware scan returned an error."
	case "com.clario360.file.expired":
		return "The linked file expired under lifecycle retention policy and should be reviewed for legal record integrity."
	default:
		return "The linked file triggered a file-integrity event."
	}
}
