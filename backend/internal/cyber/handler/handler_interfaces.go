package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/shadow"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/service"
	predictdto "github.com/clario360/platform/internal/cyber/vciso/predict/dto"
)

// dashboardService abstracts the dashboard service layer for testability.
type dashboardService interface {
	GetSOCDashboard(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, error)
	GetKPIs(ctx context.Context, tenantID uuid.UUID) (model.KPICards, error)
	GetAlertTimeline(ctx context.Context, tenantID uuid.UUID) (model.AlertTimelineData, error)
	GetSeverityDistribution(ctx context.Context, tenantID uuid.UUID) (model.SeverityDistribution, error)
	GetMTTR(ctx context.Context, tenantID uuid.UUID) (*model.MTTRReport, error)
	GetAnalystWorkload(ctx context.Context, tenantID uuid.UUID) ([]model.AnalystWorkloadEntry, error)
	GetTopAttackedAssets(ctx context.Context, tenantID uuid.UUID) ([]model.AssetAlertSummary, error)
	GetMITREHeatmap(ctx context.Context, tenantID uuid.UUID) (model.MITREHeatmapData, error)
	GetMetrics(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardMetricsResponse, error)
	GetTrends(ctx context.Context, tenantID uuid.UUID, days int) (*dto.DashboardTrendsResponse, error)
}

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

// threatFeedService abstracts the threat-feed service layer for testability.
type threatFeedService interface {
	ListFeeds(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, sort, order string, actor *service.Actor) (*dto.ThreatFeedListResponse, error)
	GetFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (*model.ThreatFeedConfig, error)
	CreateFeed(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error)
	UpdateFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error)
	DeleteFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) error
	SyncFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (map[string]interface{}, error)
	ListHistory(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) ([]*model.ThreatFeedSyncHistory, error)
}

// dspmService abstracts the DSPM service layer for testability.
type dspmService interface {
	ListDataAssets(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error)
	GetDataAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error)
	TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.DSPMScanTriggerRequest) (*model.DSPMScan, error)
	ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error)
	GetScan(ctx context.Context, tenantID, scanID uuid.UUID) (*model.DSPMScanResult, error)
	ClassificationSummary(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error)
	ExposureAnalysis(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error)
	Dependencies(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error)
	Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error)
	DetectShadowCopies(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error)
}

// riskService abstracts the risk service layer for testability.
type riskService interface {
	GetCurrentScore(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error)
	Recalculate(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.OrganizationRiskScore, error)
	Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.RiskTrendPoint, error)
	Heatmap(ctx context.Context, tenantID uuid.UUID) (*model.RiskHeatmap, error)
	TopRisks(ctx context.Context, tenantID uuid.UUID) ([]model.RiskContributor, error)
	Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error)
}

// alertService abstracts the alert service layer for testability.
type alertService interface {
	ListAlerts(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertListResponse, error)
	GetAlert(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) (*model.Alert, error)
	UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertStatusUpdateRequest) (*model.Alert, error)
	Assign(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, assignedTo uuid.UUID) (*model.Alert, error)
	Escalate(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, escalatedTo uuid.UUID, reason string) (*model.Alert, error)
	MarkFalsePositive(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, reason string) (*model.Alert, error)
	AddComment(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertCommentRequest) (*model.AlertComment, error)
	ListComments(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertComment, error)
	ListTimeline(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertTimelineEntry, error)
	Merge(ctx context.Context, tenantID, primaryAlertID uuid.UUID, mergeIDs []uuid.UUID, actor *service.Actor) (*model.Alert, error)
	Related(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.Alert, error)
	Stats(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.AlertStats, error)
	Count(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (int, error)
	BulkUpdateStatus(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertStatusRequest) (*dto.BulkOperationResult, error)
	BulkAssign(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertAssignRequest) (*dto.BulkOperationResult, error)
	BulkMarkFalsePositive(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertFalsePositiveRequest) (*dto.BulkOperationResult, error)
	CountWithHistory(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertCountResponse, error)
}

// mitreRuleService abstracts the rule service methods used by the MITRE handler.
type mitreRuleService interface {
	Coverage(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) ([]dto.MITRECoverageDTO, error)
	TechniqueDetail(ctx context.Context, tenantID uuid.UUID, techniqueID string, actor *service.Actor) (*dto.MITRETechniqueDetailDTO, error)
}

// vcisoService abstracts the vCISO service layer for testability.
type vcisoService interface {
	GenerateBriefing(ctx context.Context, tenantID, userID uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error)
	ListBriefings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error)
	Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error)
	GenerateReport(ctx context.Context, tenantID, userID uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error)
	PostureSummary(ctx context.Context, tenantID uuid.UUID) (*model.PostureSummary, error)
}

// analyticsForecastEngine abstracts the forecast engine for testability.
type analyticsForecastEngine interface {
	PredictTechniqueTrends(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.TechniqueTrendResponse, error)
	ForecastAlertVolume(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.ForecastResponse, error)
	DetectCampaigns(ctx context.Context, tenantID uuid.UUID, lookbackDays int) (*predictdto.CampaignResponse, error)
}

// threatStatsProvider abstracts the threat stats query for testability.
type threatStatsProvider interface {
	Stats(ctx context.Context, tenantID uuid.UUID) (*model.ThreatStats, error)
}
