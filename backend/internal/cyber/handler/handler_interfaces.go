package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/shadow"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/service"
)

// remediationService abstracts the remediation service layer for testability.
type remediationService interface {
	Create(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.CreateRemediationRequest) (*model.RemediationAction, error)
	List(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) (*dto.RemediationListResponse, error)
	Get(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationAction, error)
	Update(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.UpdateRemediationRequest) (*model.RemediationAction, error)
	Delete(ctx context.Context, tenantID, remediationID uuid.UUID, actor *service.Actor) error
	Submit(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error)
	Approve(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ApproveRemediationRequest) (*model.RemediationAction, error)
	Reject(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error)
	RequestRevision(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RequestRevisionRequest) (*model.RemediationAction, error)
	DryRun(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.DryRunResult, error)
	GetDryRun(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.DryRunResult, error)
	Execute(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ExecuteRemediationRequest) (*model.RemediationAction, error)
	Verify(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.VerifyRemediationRequest) (*model.RemediationAction, error)
	Rollback(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RollbackRequest) (*model.RemediationAction, error)
	Close(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error)
	AuditTrail(ctx context.Context, tenantID, remediationID uuid.UUID) ([]model.RemediationAuditEntry, error)
	Stats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error)
}

// dspmService abstracts the DSPM service layer for testability.
type dspmService interface {
	ListDataAssets(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error)
	GetDataAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error)
	TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor) (*model.DSPMScan, error)
	ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error)
	GetScan(ctx context.Context, tenantID, scanID uuid.UUID) (*model.DSPMScanResult, error)
	ClassificationSummary(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error)
	ExposureAnalysis(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error)
	Dependencies(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error)
	Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error)
	DetectShadowCopies(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error)
}

// vcisoService abstracts the vCISO service layer for testability.
type vcisoService interface {
	GenerateBriefing(ctx context.Context, tenantID, userID uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error)
	ListBriefings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error)
	Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error)
	GenerateReport(ctx context.Context, tenantID, userID uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error)
	PostureSummary(ctx context.Context, tenantID uuid.UUID) (*model.PostureSummary, error)
}
