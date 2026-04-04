package dspm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// DSPMScanner discovers data assets and evaluates posture and sensitivity.
type DSPMScanner struct {
	db         *pgxpool.Pool
	repo       *repository.DSPMRepository
	classifier *DSPMClassifier
	posture    *PostureAssessor
	dependency *DependencyMapper
	logger     zerolog.Logger
}

// NewDSPMScanner creates a DSPM scanner.
func NewDSPMScanner(
	db *pgxpool.Pool,
	repo *repository.DSPMRepository,
	classifier *DSPMClassifier,
	posture *PostureAssessor,
	dependency *DependencyMapper,
	logger zerolog.Logger,
) *DSPMScanner {
	return &DSPMScanner{
		db:         db,
		repo:       repo,
		classifier: classifier,
		posture:    posture,
		dependency: dependency,
		logger:     logger.With().Str("component", "dspm-scanner").Logger(),
	}
}

// ScanOptions holds optional parameters sent from the scan trigger request.
type ScanOptions struct {
	Scope      []string // e.g. ["databases", "cloud_storage", "file_servers", "api_endpoints"]
	AssetTypes string   // comma-separated asset type filter, e.g. "postgresql,mysql"
	FullScan   bool     // override cached results
}

// ScanWithOptions is like Scan but applies optional scope filters from the trigger request.
func (s *DSPMScanner) ScanWithOptions(ctx context.Context, tenantID uuid.UUID, scan *model.DSPMScan, opts *ScanOptions) (*model.DSPMScanResult, error) {
	start := time.Now()
	assets, err := s.discoverAssets(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Apply scope and asset_type filters from the trigger request.
	if opts != nil {
		assets = filterAssetsByOptions(assets, opts)
	}

	return s.scanAssets(ctx, tenantID, scan, assets, start)
}

// Scan evaluates all relevant tenant assets and persists DSPM records.
func (s *DSPMScanner) Scan(ctx context.Context, tenantID uuid.UUID, scan *model.DSPMScan) (*model.DSPMScanResult, error) {
	return s.ScanWithOptions(ctx, tenantID, scan, nil)
}

// filterAssetsByOptions filters discovered assets using the scope and asset type options.
func filterAssetsByOptions(assets []*model.Asset, opts *ScanOptions) []*model.Asset {
	if opts == nil {
		return assets
	}
	// Build scope-to-asset-type mapping.
	scopeTypeMap := map[string]string{
		"databases":     "database",
		"cloud_storage": "cloud_resource",
		"file_servers":  "application",
		"api_endpoints": "application",
	}
	var allowedTypes map[string]bool
	if len(opts.Scope) > 0 {
		allowedTypes = make(map[string]bool)
		for _, scope := range opts.Scope {
			if t, ok := scopeTypeMap[scope]; ok {
				allowedTypes[t] = true
			}
		}
	}

	// Parse comma-separated asset_types filter if provided.
	var nameFilter map[string]bool
	if opts.AssetTypes != "" {
		nameFilter = make(map[string]bool)
		for _, t := range strings.Split(opts.AssetTypes, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				nameFilter[strings.ToLower(t)] = true
			}
		}
	}

	filtered := make([]*model.Asset, 0, len(assets))
	for _, asset := range assets {
		if allowedTypes != nil && !allowedTypes[string(asset.Type)] {
			continue
		}
		if nameFilter != nil {
			dbType := strings.ToLower(string(asset.Type))
			// Try to extract database_type from raw JSON metadata.
			if len(asset.Metadata) > 0 {
				var meta map[string]interface{}
				if err := json.Unmarshal(asset.Metadata, &meta); err == nil {
					if dt, ok := meta["database_type"].(string); ok && dt != "" {
						dbType = strings.ToLower(dt)
					}
				}
			}
			if !nameFilter[dbType] {
				continue
			}
		}
		filtered = append(filtered, asset)
	}
	return filtered
}

func (s *DSPMScanner) scanAssets(ctx context.Context, tenantID uuid.UUID, scan *model.DSPMScan, assets []*model.Asset, start time.Time) (*model.DSPMScanResult, error) {
	var piiAssetsFound, highRiskFound, findingsCount int
	now := time.Now().UTC()
	for _, asset := range assets {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		classification := s.classifier.Classify(asset)
		posture, err := s.posture.Assess(ctx, asset, classification)
		if err != nil {
			return nil, fmt.Errorf("assess posture for %s: %w", asset.ID, err)
		}

		consumerCount, producerCount, err := s.dependency.Counts(ctx, tenantID, asset.ID)
		if err != nil {
			return nil, fmt.Errorf("dependency counts for %s: %w", asset.ID, err)
		}

		riskScore, riskFactors := CalculateRiskScore(classification.SensitivityScore, derefString(posture.NetworkExposure), posture.Score)
		dataAsset := &model.DSPMDataAsset{
			TenantID:             tenantID,
			AssetID:              asset.ID,
			AssetName:            asset.Name,
			AssetType:            string(asset.Type),
			ScanID:               &scan.ID,
			DataClassification:   classification.Classification,
			SensitivityScore:     classification.SensitivityScore,
			ContainsPII:          classification.ContainsPII,
			PIITypes:             classification.PIITypes,
			PIIColumnCount:       classification.PIIColumnCount,
			EstimatedRecordCount: posture.RecordCount,
			EncryptedAtRest:      posture.EncryptedAtRest,
			EncryptedInTransit:   posture.EncryptedInTransit,
			AccessControlType:    posture.AccessControlType,
			NetworkExposure:      posture.NetworkExposure,
			BackupConfigured:     posture.BackupConfigured,
			AuditLogging:         posture.AuditLogging,
			LastAccessReview:     posture.LastAccessReview,
			RiskScore:            riskScore,
			RiskFactors:          riskFactors,
			PostureScore:         posture.Score,
			PostureFindings:      posture.Findings,
			ConsumerCount:        consumerCount,
			ProducerCount:        producerCount,
			DatabaseType:         posture.DatabaseType,
			SchemaInfo:           posture.SchemaInfo,
			Metadata:             decodeAssetMetadata(asset),
			LastScannedAt:        &now,
		}
		if err := s.repo.UpsertDataAsset(ctx, dataAsset); err != nil {
			return nil, fmt.Errorf("persist dspm asset %s: %w", asset.ID, err)
		}

		if classification.ContainsPII {
			piiAssetsFound++
		}
		if riskScore >= 70 {
			highRiskFound++
		}
		findingsCount += len(posture.Findings)
	}

	durationMs := time.Since(start).Milliseconds()
	if err := s.repo.UpdateScanCompleted(ctx, tenantID, scan.ID, len(assets), piiAssetsFound, highRiskFound, findingsCount, durationMs); err != nil {
		return nil, fmt.Errorf("mark dspm scan completed: %w", err)
	}
	completed, err := s.repo.GetScanByID(ctx, tenantID, scan.ID)
	if err != nil {
		return nil, err
	}
	return &model.DSPMScanResult{
		Scan:           completed,
		AssetsScanned:  len(assets),
		PIIAssetsFound: piiAssetsFound,
		HighRiskFound:  highRiskFound,
		FindingsCount:  findingsCount,
	}, nil
}

func (s *DSPMScanner) discoverAssets(ctx context.Context, tenantID uuid.UUID) ([]*model.Asset, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, name, type::text, host(ip_address), hostname, mac_address::text,
		       os, os_version, owner, department, location, criticality::text, status::text,
		       discovered_at, last_seen_at, discovery_source, metadata, tags, created_by, created_at, updated_at
		FROM assets
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND (
			  type = 'database'
			  OR (type = 'application' AND (
				   metadata ? 'schema_info'
				   OR metadata ? 'columns'
				   OR tags && ARRAY['data', 'data-store', 'storage']
			  ))
			  OR (type = 'cloud_resource' AND tags && ARRAY['s3', 'blob', 'gcs', 'storage'])
		  )
		ORDER BY criticality DESC, name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("discover data assets: %w", err)
	}
	defer rows.Close()

	assets := make([]*model.Asset, 0)
	for rows.Next() {
		var (
			asset   model.Asset
			typeStr string
			critStr string
			statStr string
		)
		if err := rows.Scan(
			&asset.ID,
			&asset.TenantID,
			&asset.Name,
			&typeStr,
			&asset.IPAddress,
			&asset.Hostname,
			&asset.MACAddress,
			&asset.OS,
			&asset.OSVersion,
			&asset.Owner,
			&asset.Department,
			&asset.Location,
			&critStr,
			&statStr,
			&asset.DiscoveredAt,
			&asset.LastSeenAt,
			&asset.DiscoverySource,
			&asset.Metadata,
			&asset.Tags,
			&asset.CreatedBy,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, err
		}
		asset.Type = model.AssetType(strings.ToLower(typeStr))
		asset.Criticality = model.Criticality(strings.ToLower(critStr))
		asset.Status = model.AssetStatus(strings.ToLower(statStr))
		if asset.Tags == nil {
			asset.Tags = []string{}
		}
		assets = append(assets, &asset)
	}
	return assets, rows.Err()
}

func derefString(value *string) string {
	if value == nil {
		return "internal_only"
	}
	return *value
}
