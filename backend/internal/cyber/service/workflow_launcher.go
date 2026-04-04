package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	workflowdto "github.com/clario360/platform/internal/workflow/dto"
	workflowexecutor "github.com/clario360/platform/internal/workflow/executor"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
	workflowservice "github.com/clario360/platform/internal/workflow/service"
)

type WorkflowRemediationLauncher struct {
	defRepo     *workflowrepo.DefinitionRepository
	instRepo    *workflowrepo.InstanceRepository
	engine      *workflowservice.EngineService
	templateSvc *workflowservice.TemplateService
	logger      zerolog.Logger
}

func NewWorkflowRemediationLauncher(
	defRepo *workflowrepo.DefinitionRepository,
	instRepo *workflowrepo.InstanceRepository,
	taskRepo *workflowrepo.TaskRepository,
	logger zerolog.Logger,
) *WorkflowRemediationLauncher {
	execRegistry := workflowexecutor.NewExecutorRegistry()
	execRegistry.Register("human_task", workflowexecutor.NewHumanTaskExecutor(taskRepo, nil, logger))
	engine := workflowservice.NewEngineService(instRepo, defRepo, taskRepo, execRegistry, nil, logger)
	return &WorkflowRemediationLauncher{
		defRepo:     defRepo,
		instRepo:    instRepo,
		engine:      engine,
		templateSvc: workflowservice.NewTemplateService(defRepo, logger),
		logger:      logger.With().Str("component", "workflow-remediation-launcher").Logger(),
	}
}

func (l *WorkflowRemediationLauncher) StartRemediation(ctx context.Context, tenantID, userID uuid.UUID, group *cybermodel.CTEMRemediationGroup, assessment *cybermodel.CTEMAssessment) (string, error) {
	definitionID, err := l.ensureActiveChangeRequestDefinition(ctx, tenantID.String(), userID.String())
	if err != nil {
		return "", err
	}

	groupPriority := "medium"
	switch group.PriorityGroup {
	case 1:
		groupPriority = "critical"
	case 2:
		groupPriority = "high"
	case 3:
		groupPriority = "medium"
	default:
		groupPriority = "low"
	}

	inst, err := l.engine.StartInstance(ctx, tenantID.String(), userID.String(), workflowdto.StartInstanceRequest{
		DefinitionID: definitionID,
		InputVariables: map[string]interface{}{
			"change_id":        group.ID.String(),
			"change_type":      "security_remediation",
			"priority":         groupPriority,
			"submitter_id":     userID.String(),
			"affected_systems": assessment.ResolvedAssetIDs,
		},
	})
	if err != nil {
		return "", fmt.Errorf("start remediation workflow: %w", err)
	}
	return inst.ID, nil
}

func (l *WorkflowRemediationLauncher) ensureActiveChangeRequestDefinition(ctx context.Context, tenantID, userID string) (string, error) {
	defs, _, err := l.defRepo.List(ctx, tenantID, "active", "Change Request", "", "", "", 10, 0)
	if err != nil {
		return "", err
	}
	for _, def := range defs {
		if def.Name == "Change Request" {
			return def.ID, nil
		}
	}
	def, err := l.templateSvc.InstantiateTemplate(ctx, tenantID, userID, "tmpl-change-request", "", "")
	if err != nil {
		return "", err
	}
	def.Status = "active"
	def.UpdatedBy = userID
	if err := l.defRepo.Update(ctx, def); err != nil {
		return "", err
	}
	return def.ID, nil
}
