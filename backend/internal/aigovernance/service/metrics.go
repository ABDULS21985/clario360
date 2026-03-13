package service

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	ModelsTotal               *prometheus.GaugeVec
	ModelVersionsTotal        *prometheus.GaugeVec
	PredictionsTotal          *prometheus.CounterVec
	PredictionLatencySeconds  *prometheus.HistogramVec
	PredictionConfidence      *prometheus.HistogramVec
	PredictionLogsQueued      prometheus.Gauge
	PredictionLogsDropped     prometheus.Counter
	PredictionLogsWritten     prometheus.Counter
	PredictionFeedbackTotal   *prometheus.CounterVec
	ShadowExecutionsTotal     *prometheus.CounterVec
	ShadowDivergencesTotal    *prometheus.CounterVec
	ShadowAgreementRate       *prometheus.GaugeVec
	DriftPSI                  *prometheus.GaugeVec
	DriftAlertsTotal          *prometheus.CounterVec
	LifecyclePromotionsTotal  *prometheus.CounterVec
	LifecycleRollbacksTotal   *prometheus.CounterVec

	// Monitoring alert metrics (used by clario360-alerts.yaml)
	ModelDriftScore prometheus.Gauge // clario360_ai_model_drift_score

	// Benchmark & inference server metrics.
	BenchmarkRunsTotal      *prometheus.CounterVec
	BenchmarkLatencySeconds *prometheus.HistogramVec
	InferenceServerHealth   *prometheus.GaugeVec
	ComputeCostPerToken     *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	m := &Metrics{
		ModelsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_models_total",
			Help: "Registered AI models by tenant, suite, and status.",
		}, []string{"tenant_id", "suite", "status"}),
		ModelVersionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_model_versions_total",
			Help: "Registered AI model versions by tenant and status.",
		}, []string{"tenant_id", "status"}),
		PredictionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_predictions_total",
			Help: "Total AI predictions processed.",
		}, []string{"model_slug", "suite", "is_shadow"}),
		PredictionLatencySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_prediction_latency_seconds",
			Help:    "Latency of model invocations.",
			Buckets: prometheus.ExponentialBuckets(0.0005, 2, 12),
		}, []string{"model_slug", "suite"}),
		PredictionConfidence: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_prediction_confidence",
			Help:    "Confidence distribution for AI predictions.",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1},
		}, []string{"model_slug"}),
		PredictionLogsQueued: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ai_prediction_logs_queued",
			Help: "Current depth of the async AI prediction log queue.",
		}),
		PredictionLogsDropped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "ai_prediction_logs_dropped_total",
			Help: "Prediction logs dropped because the queue was full.",
		}),
		PredictionLogsWritten: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "ai_prediction_logs_written_total",
			Help: "Prediction logs successfully written to storage.",
		}),
		PredictionFeedbackTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_prediction_feedback_total",
			Help: "Feedback submissions for AI predictions.",
		}, []string{"model_slug", "correct"}),
		ShadowExecutionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_shadow_executions_total",
			Help: "Shadow model executions launched asynchronously.",
		}, []string{"model_slug"}),
		ShadowDivergencesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_shadow_divergences_total",
			Help: "Shadow predictions that diverged from production.",
		}, []string{"model_slug"}),
		ShadowAgreementRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_shadow_agreement_rate",
			Help: "Latest shadow agreement rate per model.",
		}, []string{"model_slug"}),
		DriftPSI: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_drift_psi",
			Help: "Latest PSI score per model.",
		}, []string{"model_slug"}),
		DriftAlertsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_drift_alerts_total",
			Help: "Drift alerts emitted by model and drift level.",
		}, []string{"model_slug", "drift_level"}),
		LifecyclePromotionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_lifecycle_promotions_total",
			Help: "Lifecycle promotions by model and target status.",
		}, []string{"model_slug", "to_status"}),
		LifecycleRollbacksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_lifecycle_rollbacks_total",
			Help: "Lifecycle rollbacks by model.",
		}, []string{"model_slug"}),
		ModelDriftScore: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "clario360_ai_model_drift_score",
			Help: "Maximum model drift score across all active models (used by Prometheus alerting rules).",
		}),
		BenchmarkRunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_benchmark_runs_total",
			Help: "Benchmark runs by backend type and status.",
		}, []string{"backend_type", "status"}),
		BenchmarkLatencySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_benchmark_latency_seconds",
			Help:    "Observed latency distribution in benchmark runs.",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 14),
		}, []string{"backend_type"}),
		InferenceServerHealth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_inference_server_health",
			Help: "Health status of inference servers (1=healthy, 0=unhealthy).",
		}, []string{"server_name", "backend_type"}),
		ComputeCostPerToken: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_compute_cost_per_token",
			Help: "Estimated cost per 1K tokens by backend.",
		}, []string{"backend_type"}),
	}
	reg.MustRegister(
		m.ModelsTotal,
		m.ModelVersionsTotal,
		m.PredictionsTotal,
		m.PredictionLatencySeconds,
		m.PredictionConfidence,
		m.PredictionLogsQueued,
		m.PredictionLogsDropped,
		m.PredictionLogsWritten,
		m.PredictionFeedbackTotal,
		m.ShadowExecutionsTotal,
		m.ShadowDivergencesTotal,
		m.ShadowAgreementRate,
		m.DriftPSI,
		m.DriftAlertsTotal,
		m.LifecyclePromotionsTotal,
		m.LifecycleRollbacksTotal,
		m.ModelDriftScore,
		m.BenchmarkRunsTotal,
		m.BenchmarkLatencySeconds,
		m.InferenceServerHealth,
		m.ComputeCostPerToken,
	)
	return m
}
