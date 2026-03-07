package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type DSPMService struct {
	repo       *repository.DSPMRepository
	scanner    *dspm.DSPMScanner
	dependency *dspm.DependencyMapper
	producer   *events.Producer
	logger     zerolog.Logger

	mu      sync.Mutex
	running map[uuid.UUID]context.CancelFunc
}

func NewDSPMService(
	repo *repository.DSPMRepository,
	scanner *dspm.DSPMScanner,
	dependency *dspm.DependencyMapper,
	producer *events.Producer,
	logger zerolog.Logger,
) *DSPMService {
	return &DSPMService{
		repo:       repo,
		scanner:    scanner,
		dependency: dependency,
		producer:   producer,
		logger:     logger.With().Str("service", "dspm").Logger(),
		running:    make(map[uuid.UUID]context.CancelFunc),
	}
}

func (s *DSPMService) ListDataAssets(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListDataAssets(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages == 0 {
		totalPages = 1
	}
	return &dto.DSPMAssetListResponse{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *DSPMService) GetDataAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error) {
	return s.repo.GetDataAssetByID(ctx, tenantID, assetID)
}

func (s *DSPMService) TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor) (*model.DSPMScan, error) {
	scan, err := s.repo.CreateScan(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if err := publishEvent(ctx, s.producer, events.Topics.DSPMEvents, "com.clario360.cyber.dspm.scan_started", tenantID, actor, map[string]interface{}{
		"scan_id":   scan.ID.String(),
		"tenant_id": tenantID.String(),
	}); err != nil {
		s.logger.Error().Err(err).Str("scan_id", scan.ID.String()).Msg("publish dspm scan started event")
	}

	runCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	s.mu.Lock()
	s.running[scan.ID] = cancel
	s.mu.Unlock()

	go func(scanID uuid.UUID) {
		defer func() {
			cancel()
			s.mu.Lock()
			delete(s.running, scanID)
			s.mu.Unlock()
		}()

		result, err := s.scanner.Scan(runCtx, tenantID, scan)
		if err != nil {
			s.logger.Error().Err(err).Str("scan_id", scanID.String()).Msg("dspm scan failed")
			_ = s.repo.UpdateScanFailed(context.Background(), tenantID, scanID)
			return
		}
		if err := publishEvent(context.Background(), s.producer, events.Topics.DSPMEvents, "com.clario360.cyber.dspm.scan_completed", tenantID, actor, map[string]interface{}{
			"scan_id":        scanID.String(),
			"assets_scanned": result.AssetsScanned,
			"pii_found":      result.PIIAssetsFound,
			"high_risk":      result.HighRiskFound,
		}); err != nil {
			s.logger.Error().Err(err).Str("scan_id", scanID.String()).Msg("publish dspm scan completed event")
		}
	}(scan.ID)

	return scan, nil
}

func (s *DSPMService) ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListScans(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages == 0 {
		totalPages = 1
	}
	return &dto.DSPMScanListResponse{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *DSPMService) GetScan(ctx context.Context, tenantID, scanID uuid.UUID) (*model.DSPMScanResult, error) {
	scan, err := s.repo.GetScanByID(ctx, tenantID, scanID)
	if err != nil {
		return nil, err
	}
	return &model.DSPMScanResult{
		Scan:           scan,
		AssetsScanned:  scan.AssetsScanned,
		PIIAssetsFound: scan.PIIAssetsFound,
		HighRiskFound:  scan.HighRiskFound,
		FindingsCount:  scan.FindingsCount,
	}, nil
}

func (s *DSPMService) ClassificationSummary(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error) {
	dashboard, err := s.repo.Dashboard(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &model.DSPMClassificationSummary{
		Public:       dashboard.ClassificationBreakdown["public"],
		Internal:     dashboard.ClassificationBreakdown["internal"],
		Confidential: dashboard.ClassificationBreakdown["confidential"],
		Restricted:   dashboard.ClassificationBreakdown["restricted"],
		Total:        dashboard.TotalDataAssets,
	}, nil
}

func (s *DSPMService) ExposureAnalysis(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error) {
	dashboard, err := s.repo.Dashboard(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	params := &dto.DSPMAssetListParams{
		Page:            1,
		PerPage:         100,
		Sort:            "risk_score",
		Order:           "desc",
		NetworkExposure: nullableString("internet_facing"),
	}
	params.SetDefaults()
	assets, _, err := s.repo.ListDataAssets(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}

	critical := make([]model.DSPMDataAsset, 0, len(assets))
	for _, asset := range assets {
		if asset.RiskScore >= 70 || asset.DataClassification == "restricted" || asset.DataClassification == "confidential" {
			critical = append(critical, *asset)
		}
		if len(critical) == 10 {
			break
		}
	}

	return &model.DSPMExposureAnalysis{
		InternalOnly:      dashboard.ExposureBreakdown["internal_only"],
		VPNAccessible:     dashboard.ExposureBreakdown["vpn_accessible"],
		InternetFacing:    dashboard.ExposureBreakdown["internet_facing"],
		Unknown:           dashboard.ExposureBreakdown["unknown"],
		CriticalExposures: critical,
	}, nil
}

func (s *DSPMService) Dependencies(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error) {
	return s.dependency.MapGraph(ctx, tenantID)
}

func (s *DSPMService) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error) {
	return s.repo.Dashboard(ctx, tenantID)
}
