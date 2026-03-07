package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

type CreateCTEMAssessmentRequest struct {
	Name         string                 `json:"name" validate:"required,min=2,max=255"`
	Description  string                 `json:"description,omitempty" validate:"omitempty,max=5000"`
	Scope        model.AssessmentScope  `json:"scope" validate:"required"`
	Scheduled    bool                   `json:"scheduled"`
	ScheduleCron *string                `json:"schedule_cron,omitempty" validate:"omitempty,max=100"`
	Tags         []string               `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
	Start        bool                   `json:"start"`
}

type UpdateCTEMAssessmentRequest struct {
	Name         *string                `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description  *string                `json:"description,omitempty" validate:"omitempty,max=5000"`
	Scope        *model.AssessmentScope `json:"scope,omitempty"`
	Scheduled    *bool                  `json:"scheduled,omitempty"`
	ScheduleCron *string                `json:"schedule_cron,omitempty" validate:"omitempty,max=100"`
	Tags         *[]string              `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
}

type CTEMAssessmentListParams struct {
	Status    *string `form:"status"`
	Scheduled *bool   `form:"scheduled"`
	Search    *string `form:"search"`
	Tag       *string `form:"tag"`
	Page      int     `form:"page"`
	PerPage   int     `form:"per_page"`
	Sort      string  `form:"sort" validate:"omitempty,oneof=created_at updated_at started_at completed_at exposure_score name"`
	Order     string  `form:"order" validate:"omitempty,oneof=asc desc"`
}

func (p *CTEMAssessmentListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *CTEMAssessmentListParams) Validate() error {
	if p.Status != nil && !model.CTEMAssessmentStatus(*p.Status).IsValid() {
		return fmt.Errorf("invalid status: %q", *p.Status)
	}
	if p.PerPage < 1 || p.PerPage > 200 {
		return fmt.Errorf("per_page must be between 1 and 200")
	}
	if p.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	return nil
}

type CTEMAssessmentListResponse struct {
	Data       []*model.CTEMAssessment `json:"data"`
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	PerPage    int                     `json:"per_page"`
	TotalPages int                     `json:"total_pages"`
}

type CTEMFindingsListParams struct {
	Severity      *string `form:"severity"`
	Type          *string `form:"type"`
	Status        *string `form:"status"`
	PriorityGroup *int    `form:"priority_group"`
	Search        *string `form:"search"`
	Page          int     `form:"page"`
	PerPage       int     `form:"per_page"`
	Sort          string  `form:"sort" validate:"omitempty,oneof=priority_score priority_rank severity created_at updated_at"`
	Order         string  `form:"order" validate:"omitempty,oneof=asc desc"`
}

func (p *CTEMFindingsListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "priority_score"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *CTEMFindingsListParams) Validate() error {
	if p.Severity != nil && !model.Severity(*p.Severity).IsValid() {
		return fmt.Errorf("invalid severity: %q", *p.Severity)
	}
	if p.Type != nil && !model.CTEMFindingType(*p.Type).IsValid() {
		return fmt.Errorf("invalid type: %q", *p.Type)
	}
	if p.Status != nil {
		switch model.CTEMFindingStatus(*p.Status) {
		case model.CTEMFindingStatusOpen, model.CTEMFindingStatusInRemediation, model.CTEMFindingStatusRemediated,
			model.CTEMFindingStatusAcceptedRisk, model.CTEMFindingStatusFalsePositive, model.CTEMFindingStatusDeferred:
		default:
			return fmt.Errorf("invalid status: %q", *p.Status)
		}
	}
	if p.PriorityGroup != nil && (*p.PriorityGroup < 1 || *p.PriorityGroup > 4) {
		return fmt.Errorf("priority_group must be between 1 and 4")
	}
	return nil
}

type CTEMFindingsListResponse struct {
	Data       []*model.CTEMFinding `json:"data"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int                  `json:"total_pages"`
}

type UpdateCTEMFindingStatusRequest struct {
	Status model.CTEMFindingStatus `json:"status" validate:"required,oneof=open in_remediation remediated accepted_risk false_positive deferred"`
	Notes  *string                 `json:"notes,omitempty" validate:"omitempty,max=5000"`
}

type ValidationFindingOverride struct {
	FindingID            uuid.UUID                    `json:"finding_id" validate:"required"`
	ValidationStatus     model.CTEMValidationStatus   `json:"validation_status" validate:"required,oneof=pending validated compensated not_exploitable requires_manual"`
	ValidationNotes      *string                      `json:"validation_notes,omitempty" validate:"omitempty,max=5000"`
	CompensatingControls []string                     `json:"compensating_controls,omitempty" validate:"omitempty,max=20,dive,min=1,max=255"`
}

type ValidateAssessmentRequest struct {
	Findings []ValidationFindingOverride `json:"findings,omitempty"`
}

type UpdateCTEMRemediationGroupStatusRequest struct {
	Status model.CTEMRemediationGroupStatus `json:"status" validate:"required,oneof=planned in_progress completed deferred accepted"`
}

type CTEMReportExportRequest struct {
	Format string `json:"format" validate:"required,oneof=pdf docx"`
}

type CTEMReportExportResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type ExposureScoreHistoryParams struct {
	Days int `form:"days"`
}

func (p *ExposureScoreHistoryParams) SetDefaults() {
	if p.Days == 0 {
		p.Days = 90
	}
}

func (p *ExposureScoreHistoryParams) Since(now time.Time) time.Time {
	p.SetDefaults()
	return now.AddDate(0, 0, -p.Days)
}
