package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// FileMetrics holds all Prometheus metrics for the file service.
type FileMetrics struct {
	UploadsTotal              *prometheus.CounterVec
	UploadSizeBytes           *prometheus.HistogramVec
	UploadDuration            *prometheus.HistogramVec
	DownloadsTotal            *prometheus.CounterVec
	DownloadDuration          *prometheus.HistogramVec
	DeletesTotal              *prometheus.CounterVec
	PresignedURLsGenerated    *prometheus.CounterVec
	VirusScansTotal           *prometheus.CounterVec
	VirusScanDuration         *prometheus.HistogramVec
	QuarantinedTotal          prometheus.Counter
	EncryptionDuration        *prometheus.HistogramVec
	ContentTypeMismatchTotal  *prometheus.CounterVec
	BlockedUploadTotal        *prometheus.CounterVec
	LifecycleExpiredTotal     prometheus.Counter
	LifecyclePurgedTotal      prometheus.Counter
	StorageUsageBytes         *prometheus.GaugeVec
	MinIOErrorsTotal          *prometheus.CounterVec
	ClamAVErrorsTotal         prometheus.Counter
	AccessLogEntriesTotal     *prometheus.CounterVec
	DuplicateDetectedTotal    *prometheus.CounterVec
}

// NewFileMetrics creates and registers all file service metrics.
func NewFileMetrics(reg *prometheus.Registry) *FileMetrics {
	m := &FileMetrics{
		UploadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_uploads_total",
			Help: "Total number of file uploads",
		}, []string{"suite", "encrypted", "lifecycle"}),

		UploadSizeBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "file_upload_size_bytes",
			Help:    "Size of uploaded files in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 52428800, 104857600},
		}, []string{"suite"}),

		UploadDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "file_upload_duration_seconds",
			Help:    "Duration of file upload operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"suite", "encrypted"}),

		DownloadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_downloads_total",
			Help: "Total number of file downloads",
		}, []string{"suite"}),

		DownloadDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "file_download_duration_seconds",
			Help:    "Duration of file download operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"suite", "encrypted"}),

		DeletesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_deletes_total",
			Help: "Total number of file deletions",
		}, []string{"suite"}),

		PresignedURLsGenerated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_presigned_urls_generated_total",
			Help: "Total presigned URLs generated",
		}, []string{"type", "suite"}),

		VirusScansTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_virus_scans_total",
			Help: "Total number of virus scans",
		}, []string{"status"}),

		VirusScanDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "file_virus_scan_duration_seconds",
			Help:    "Duration of virus scans",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}, nil),

		QuarantinedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "file_quarantined_total",
			Help: "Total files quarantined due to virus detection",
		}),

		EncryptionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "file_encryption_duration_seconds",
			Help:    "Duration of encryption operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),

		ContentTypeMismatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_content_type_mismatch_total",
			Help: "Total content type mismatches detected",
		}, []string{"declared", "detected"}),

		BlockedUploadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_blocked_upload_total",
			Help: "Total uploads blocked",
		}, []string{"reason"}),

		LifecycleExpiredTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "file_lifecycle_expired_total",
			Help: "Total files expired by lifecycle",
		}),

		LifecyclePurgedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "file_lifecycle_purged_total",
			Help: "Total files purged (hard-deleted)",
		}),

		StorageUsageBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "file_storage_usage_bytes",
			Help: "Storage usage in bytes",
		}, []string{"tenant_id", "suite"}),

		MinIOErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_minio_errors_total",
			Help: "Total MinIO operation errors",
		}, []string{"operation"}),

		ClamAVErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "file_clamav_errors_total",
			Help: "Total ClamAV errors",
		}),

		AccessLogEntriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_access_log_entries_total",
			Help: "Total access log entries",
		}, []string{"action"}),

		DuplicateDetectedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "file_duplicate_detected_total",
			Help: "Total duplicate files detected",
		}, []string{"suite"}),
	}

	reg.MustRegister(
		m.UploadsTotal,
		m.UploadSizeBytes,
		m.UploadDuration,
		m.DownloadsTotal,
		m.DownloadDuration,
		m.DeletesTotal,
		m.PresignedURLsGenerated,
		m.VirusScansTotal,
		m.VirusScanDuration,
		m.QuarantinedTotal,
		m.EncryptionDuration,
		m.ContentTypeMismatchTotal,
		m.BlockedUploadTotal,
		m.LifecycleExpiredTotal,
		m.LifecyclePurgedTotal,
		m.StorageUsageBytes,
		m.MinIOErrorsTotal,
		m.ClamAVErrorsTotal,
		m.AccessLogEntriesTotal,
		m.DuplicateDetectedTotal,
	)

	return m
}
