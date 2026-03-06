package health

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"

	"github.com/clario360/platform/internal/observability/health"
	"github.com/clario360/platform/pkg/storage"
)

// MinIOChecker implements health.HealthChecker for MinIO connectivity.
type MinIOChecker struct {
	client *minio.Client
	name   string
}

// NewMinIOChecker creates a MinIO health checker.
func NewMinIOChecker(client *minio.Client) *MinIOChecker {
	return &MinIOChecker{client: client, name: "minio"}
}

func (c *MinIOChecker) Name() string { return c.name }

func (c *MinIOChecker) Check(ctx context.Context) health.HealthResult {
	_, err := c.client.ListBuckets(ctx)
	if err != nil {
		return health.HealthResult{
			Status: "unhealthy",
			Error:  fmt.Sprintf("minio: %v", err),
		}
	}
	return health.HealthResult{
		Status: "healthy",
	}
}

// ClamAVChecker implements health.HealthChecker for ClamAV.
type ClamAVChecker struct {
	scanner *storage.VirusScanner
	name    string
}

// NewClamAVChecker creates a ClamAV health checker.
func NewClamAVChecker(scanner *storage.VirusScanner) *ClamAVChecker {
	return &ClamAVChecker{scanner: scanner, name: "clamav"}
}

func (c *ClamAVChecker) Name() string { return c.name }

func (c *ClamAVChecker) Check(ctx context.Context) health.HealthResult {
	if c.scanner == nil {
		return health.HealthResult{
			Status: "degraded",
			Error:  "clamav scanner not configured",
		}
	}

	if err := c.scanner.Ping(); err != nil {
		return health.HealthResult{
			Status: "degraded", // degraded, not unhealthy — ClamAV is optional at startup
			Error:  fmt.Sprintf("clamav: %v", err),
		}
	}

	return health.HealthResult{
		Status: "healthy",
	}
}
