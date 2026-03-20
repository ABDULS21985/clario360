package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
)

type hdfsScanner interface {
	connector.Connector
	ScanFile(ctx context.Context, path string, sampleBytes int64) (*connector.FileScanResult, error)
	ListRecentFiles(ctx context.Context, since time.Time, basePath string) ([]connector.FileInfo, error)
}

type DSPMScannerAdapter struct {
	connectorRegistry connectorRegistry
	sourceRepo        sourceRepository
	decryptor         configDecryptor
	logger            zerolog.Logger
}

func NewDSPMScannerAdapter(
	registry connectorRegistry,
	sourceRepo sourceRepository,
	decryptor configDecryptor,
	logger zerolog.Logger,
) *DSPMScannerAdapter {
	return &DSPMScannerAdapter{
		connectorRegistry: registry,
		sourceRepo:        sourceRepo,
		decryptor:         decryptor,
		logger:            logger.With().Str("component", "data-dspm-adapter").Logger(),
	}
}

func (a *DSPMScannerAdapter) ListLocations(ctx context.Context, tenantID uuid.UUID) ([]connector.DataLocation, error) {
	records, err := a.sourceRepo.ListActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list active sources for dspm adapter: %w", err)
	}

	locations := make([]connector.DataLocation, 0)
	var errs []error
	for _, record := range records {
		if record == nil || record.Source == nil {
			continue
		}
		configJSON, decryptErr := a.decryptor.Decrypt(record.EncryptedConfig)
		if decryptErr != nil {
			errs = append(errs, fmt.Errorf("decrypt source %s: %w", record.Source.ID, decryptErr))
			continue
		}
		instance, createErr := a.connectorRegistry.CreateWithSourceContext(record.Source.Type, configJSON, record.Source.ID, record.Source.TenantID)
		if createErr != nil {
			errs = append(errs, fmt.Errorf("create connector for source %s: %w", record.Source.ID, createErr))
			continue
		}
		func() {
			defer func() {
				if closeErr := instance.Close(); closeErr != nil {
					a.logger.Warn().Err(closeErr).Str("source_id", record.Source.ID.String()).Msg("close dspm adapter connector")
				}
			}()
			securityConnector, ok := instance.(connector.SecurityAwareConnector)
			if !ok {
				return
			}
			found, listErr := securityConnector.ListDataLocations(ctx)
			if listErr != nil {
				errs = append(errs, fmt.Errorf("list data locations for source %s: %w", record.Source.ID, listErr))
				return
			}
			locations = append(locations, found...)
		}()
	}
	return locations, errors.Join(errs...)
}

func (a *DSPMScannerAdapter) ScanRecentHDFSFiles(ctx context.Context, tenantID uuid.UUID, since time.Time, sampleBytes int64) ([]connector.FileScanResult, error) {
	records, err := a.sourceRepo.ListActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list active sources for hdfs scan: %w", err)
	}

	results := make([]connector.FileScanResult, 0)
	var errs []error
	for _, record := range records {
		if record == nil || record.Source == nil || record.Source.Type != model.DataSourceTypeHDFS {
			continue
		}
		configJSON, decryptErr := a.decryptor.Decrypt(record.EncryptedConfig)
		if decryptErr != nil {
			errs = append(errs, fmt.Errorf("decrypt hdfs source %s: %w", record.Source.ID, decryptErr))
			continue
		}
		instance, createErr := a.connectorRegistry.CreateWithSourceContext(record.Source.Type, configJSON, record.Source.ID, record.Source.TenantID)
		if createErr != nil {
			errs = append(errs, fmt.Errorf("create hdfs connector for source %s: %w", record.Source.ID, createErr))
			continue
		}
		func() {
			defer func() {
				if closeErr := instance.Close(); closeErr != nil {
					a.logger.Warn().Err(closeErr).Str("source_id", record.Source.ID.String()).Msg("close hdfs scan connector")
				}
			}()
			scanner, ok := instance.(hdfsScanner)
			if !ok {
				errs = append(errs, fmt.Errorf("source %s does not expose hdfs scan capabilities", record.Source.ID))
				return
			}
			files, listErr := scanner.ListRecentFiles(ctx, since, "")
			if listErr != nil {
				errs = append(errs, fmt.Errorf("list recent hdfs files for source %s: %w", record.Source.ID, listErr))
				return
			}
			for _, file := range files {
				scanResult, scanErr := scanner.ScanFile(ctx, file.Path, sampleBytes)
				if scanErr != nil {
					errs = append(errs, fmt.Errorf("scan hdfs file %s for source %s: %w", file.Path, record.Source.ID, scanErr))
					continue
				}
				if scanResult != nil {
					results = append(results, *scanResult)
				}
			}
		}()
	}
	return results, errors.Join(errs...)
}
