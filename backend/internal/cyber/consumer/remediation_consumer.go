package consumer

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

var remediationSystemActorID = uuid.MustParse("11111111-1111-4111-8111-111111111111")

type RemediationConsumer struct {
	remediationSvc  *service.RemediationService
	remediationRepo *repository.RemediationRepository
	groupRepo       *repository.CTEMRemediationGroupRepository
	findingRepo     *repository.CTEMFindingRepository
	consumer        *events.Consumer
	logger          zerolog.Logger
}

func NewRemediationConsumer(
	remediationSvc *service.RemediationService,
	remediationRepo *repository.RemediationRepository,
	groupRepo *repository.CTEMRemediationGroupRepository,
	findingRepo *repository.CTEMFindingRepository,
	consumer *events.Consumer,
	logger zerolog.Logger,
) *RemediationConsumer {
	c := &RemediationConsumer{
		remediationSvc:  remediationSvc,
		remediationRepo: remediationRepo,
		groupRepo:       groupRepo,
		findingRepo:     findingRepo,
		consumer:        consumer,
		logger:          logger.With().Str("component", "remediation-consumer").Logger(),
	}
	consumer.Subscribe(events.Topics.CtemEvents, events.EventHandlerFunc(c.handle))
	consumer.Subscribe(events.Topics.AlertEvents, events.EventHandlerFunc(c.handle))
	return c
}

func (c *RemediationConsumer) handle(ctx context.Context, event *events.Event) error {
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return err
	}

	switch event.Type {
	case "cyber.ctem.remediation.triggered", "com.clario360.cyber.ctem.remediation.triggered":
		return c.handleCTEMTriggered(ctx, tenantID, event)
	case "cyber.alert.created", "com.clario360.cyber.alert.created":
		return c.handleAlertCreated(ctx, tenantID, event)
	default:
		return nil
	}
}

func (c *RemediationConsumer) handleCTEMTriggered(ctx context.Context, tenantID uuid.UUID, event *events.Event) error {
	var payload struct {
		GroupID string `json:"group_id"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		return err
	}
	groupID, err := uuid.Parse(payload.GroupID)
	if err != nil {
		return err
	}

	if _, err := c.remediationRepo.FindActiveByRemediationGroupID(ctx, tenantID, groupID); err == nil {
		return nil
	} else if err != repository.ErrNotFound {
		return err
	}

	group, err := c.groupRepo.GetByID(ctx, tenantID, groupID)
	if err != nil {
		return err
	}
	findings, err := c.findingRepo.ListAllByAssessment(ctx, tenantID, group.AssessmentID)
	if err != nil {
		return err
	}

	assetIDs := make([]uuid.UUID, 0, group.AffectedAssetCount)
	seenAssets := make(map[uuid.UUID]struct{})
	var ctemFindingID *uuid.UUID
	for _, finding := range findings {
		if finding.RemediationGroupID == nil || *finding.RemediationGroupID != group.ID {
			continue
		}
		if ctemFindingID == nil {
			id := finding.ID
			ctemFindingID = &id
		}
		for _, assetID := range finding.AffectedAssetIDs {
			if assetID == uuid.Nil {
				continue
			}
			if _, ok := seenAssets[assetID]; ok {
				continue
			}
			seenAssets[assetID] = struct{}{}
			assetIDs = append(assetIDs, assetID)
		}
	}

	req := &dto.CreateRemediationRequest{
		AssessmentID:       &group.AssessmentID,
		CTEMFindingID:      ctemFindingID,
		RemediationGroupID: &group.ID,
		Type:               remediationTypeForGroup(group.Type),
		Severity:           remediationSeverityForPriority(group.PriorityGroup),
		Title:              group.Title,
		Description:        firstNonEmptyString(group.Description, "Generated from CTEM remediation group"),
		Plan:               remediationPlanForGroup(group, assetIDs),
		AffectedAssetIDs:   assetIDs,
		ExecutionMode:      remediationExecutionModeForGroup(group.Type),
		RequiresApprovalFrom: "security_manager",
		Tags:               []string{"ctem", "auto-generated"},
		Metadata: map[string]interface{}{
			"source":       "ctem",
			"priority_group": group.PriorityGroup,
			"ctem_group_type": string(group.Type),
		},
	}
	action, err := c.remediationSvc.Create(ctx, tenantID, remediationSystemActorID, remediationSystemActor(), req)
	if err != nil {
		return err
	}
	_, err = c.remediationSvc.Submit(ctx, tenantID, action.ID, remediationSystemActorID, remediationSystemActor().UserName, "admin")
	return err
}

func (c *RemediationConsumer) handleAlertCreated(ctx context.Context, tenantID uuid.UUID, event *events.Event) error {
	var payload struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Severity   string  `json:"severity"`
		Confidence float64 `json:"confidence"`
		AssetID    *string `json:"asset_id"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.Severity != "critical" && payload.Severity != "high" {
		return nil
	}

	alertID, err := uuid.Parse(payload.ID)
	if err != nil {
		return err
	}
	if _, err := c.remediationRepo.FindActiveByAlertID(ctx, tenantID, alertID); err == nil {
		return nil
	} else if err != repository.ErrNotFound {
		return err
	}

	assetIDs := make([]uuid.UUID, 0, 1)
	if payload.AssetID != nil && strings.TrimSpace(*payload.AssetID) != "" {
		if assetID, err := uuid.Parse(*payload.AssetID); err == nil {
			assetIDs = append(assetIDs, assetID)
		}
	}

	req := &dto.CreateRemediationRequest{
		AlertID:            &alertID,
		Type:               string(model.RemediationTypeCustom),
		Severity:           payload.Severity,
		Title:              "Investigate and remediate alert: " + payload.Title,
		Description:        "Auto-generated governed remediation from a high-priority alert. Execution remains blocked pending approval and dry-run.",
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{
				{
					Number:      1,
					Action:      "investigate_alert",
					Description: "Validate the alert, determine containment scope, and confirm remediation action with the asset owner.",
					Expected:    "Containment and remediation plan confirmed",
				},
			},
			Reversible:        true,
			EstimatedDowntime: "0",
			RiskLevel:         payload.Severity,
		},
		AffectedAssetIDs:   assetIDs,
		ExecutionMode:      "manual",
		RequiresApprovalFrom: "security_manager",
		Tags:               []string{"alert", "auto-generated"},
		Metadata: map[string]interface{}{
			"source":          "alert",
			"confidence":      payload.Confidence,
			"auto_submitted":  true,
			"governance_note": "Execution requires approval and dry-run",
		},
	}
	action, err := c.remediationSvc.Create(ctx, tenantID, remediationSystemActorID, remediationSystemActor(), req)
	if err != nil {
		return err
	}
	_, err = c.remediationSvc.Submit(ctx, tenantID, action.ID, remediationSystemActorID, remediationSystemActor().UserName, "admin")
	return err
}

func remediationSystemActor() *service.Actor {
	return &service.Actor{
		UserID:    remediationSystemActorID,
		UserName:  "cyber-remediation-consumer",
		UserEmail: "cyber-remediation-consumer@system.local",
	}
}

func remediationTypeForGroup(groupType model.CTEMRemediationType) string {
	switch groupType {
	case model.CTEMRemediationPatch, model.CTEMRemediationUpgrade:
		return string(model.RemediationTypePatch)
	case model.CTEMRemediationConfiguration:
		return string(model.RemediationTypeConfigChange)
	case model.CTEMRemediationArchitecture:
		return string(model.RemediationTypeCustom)
	case model.CTEMRemediationDecommission:
		return string(model.RemediationTypeCustom)
	case model.CTEMRemediationAcceptRisk:
		return string(model.RemediationTypeCustom)
	default:
		return string(model.RemediationTypeCustom)
	}
}

func remediationExecutionModeForGroup(groupType model.CTEMRemediationType) string {
	switch groupType {
	case model.CTEMRemediationPatch, model.CTEMRemediationConfiguration:
		return "semi_automated"
	default:
		return "manual"
	}
}

func remediationSeverityForPriority(priorityGroup int) string {
	switch priorityGroup {
	case 1:
		return "critical"
	case 2:
		return "high"
	case 3:
		return "medium"
	default:
		return "low"
	}
}

func remediationPlanForGroup(group *model.CTEMRemediationGroup, assetIDs []uuid.UUID) model.RemediationPlan {
	steps := []model.RemediationStep{
		{
			Number:      1,
			Action:      "review_scope",
			Description: "Validate affected assets, dependencies, and business impact before execution.",
			Expected:    "Scope validated",
		},
	}
	switch group.Type {
	case model.CTEMRemediationPatch, model.CTEMRemediationUpgrade:
		steps = append(steps, model.RemediationStep{
			Number:      2,
			Action:      "apply_patch",
			Description: "Apply vendor remediation to the affected assets.",
			Expected:    "Vulnerabilities mitigated pending verification",
		})
	case model.CTEMRemediationConfiguration:
		steps = append(steps, model.RemediationStep{
			Number:      2,
			Action:      "apply_config_change",
			Description: "Implement the required configuration hardening across the affected assets.",
			Expected:    "Configuration baseline enforced",
		})
	default:
		steps = append(steps, model.RemediationStep{
			Number:      2,
			Action:      "manual_remediation",
			Description: "Perform the governed remediation steps manually and document the outcome.",
			Expected:    "Manual remediation completed and ready for verification",
		})
	}

	plan := model.RemediationPlan{
		Steps:             steps,
		Reversible:        group.Type != model.CTEMRemediationAcceptRisk,
		RequiresReboot:    group.Type == model.CTEMRemediationPatch || group.Type == model.CTEMRemediationUpgrade,
		EstimatedDowntime: estimatedDowntimeForGroup(group),
		RiskLevel:         remediationSeverityForPriority(group.PriorityGroup),
		BlockTargets:      []string{},
		IsolateConfig:     map[string]interface{}{},
		TargetConfig:      map[string]interface{}{},
	}
	if len(group.CVEIDs) > 0 {
		plan.TargetVersion = group.CVEIDs[0]
	}
	if group.Type == model.CTEMRemediationArchitecture && len(assetIDs) > 0 {
		plan.IsolateConfig = map[string]interface{}{"asset_count": len(assetIDs)}
	}
	return plan
}

func estimatedDowntimeForGroup(group *model.CTEMRemediationGroup) string {
	switch group.Effort {
	case model.CTEMRemediationEffortLow:
		return "0"
	case model.CTEMRemediationEffortMedium:
		return "15m"
	default:
		return "1h"
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
