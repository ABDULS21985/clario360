package seeder

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
	workflowservice "github.com/clario360/platform/internal/workflow/service"
)

type WorkflowTemplateSeeder struct {
	definitions *workflowrepo.DefinitionRepository
	templates   *workflowservice.TemplateService
	logger      zerolog.Logger
}

func NewWorkflowTemplateSeeder(definitions *workflowrepo.DefinitionRepository, logger zerolog.Logger) *WorkflowTemplateSeeder {
	return &WorkflowTemplateSeeder{
		definitions: definitions,
		templates:   workflowservice.NewTemplateService(definitions, logger),
		logger:      logger.With().Str("component", "workflow_template_seeder").Logger(),
	}
}

func (s *WorkflowTemplateSeeder) Seed(ctx context.Context, tenantID, adminUserID string) error {
	templates, err := s.templates.ListTemplates(ctx, "")
	if err != nil {
		return err
	}
	for _, tmpl := range templates {
		definitions, _, err := s.definitions.List(ctx, tenantID, "", tmpl.Name, "", "", "", 25, 0)
		if err != nil {
			return err
		}
		exists := false
		for _, definition := range definitions {
			if definition.Name == tmpl.Name {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		if _, err := s.templates.InstantiateTemplate(ctx, tenantID, adminUserID, tmpl.ID, "", ""); err != nil {
			return fmt.Errorf("instantiate workflow template %s: %w", tmpl.ID, err)
		}
	}
	return nil
}
