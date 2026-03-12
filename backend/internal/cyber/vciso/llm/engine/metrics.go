package engine

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	QueriesTotal               *prometheus.CounterVec
	CallsTotal                 *prometheus.CounterVec
	CallLatencySeconds         *prometheus.HistogramVec
	TokensTotal                *prometheus.CounterVec
	CostUSDTotal               *prometheus.CounterVec
	ToolLoopIterations         prometheus.Histogram
	ToolCallsPerQuery          prometheus.Histogram
	GroundingResultsTotal      *prometheus.CounterVec
	InjectionDetectionsTotal   *prometheus.CounterVec
	RateLimitRejectionsTotal   *prometheus.CounterVec
	FallbackTotal              *prometheus.CounterVec
	ResponseLatencySeconds     *prometheus.HistogramVec
	ContextTokensUsed          prometheus.Histogram
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	m := &Metrics{
		QueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_queries_total",
			Help: "Total routed vCISO queries by engine and reason.",
		}, []string{"engine", "routing_reason"}),
		CallsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_calls_total",
			Help: "Total LLM provider calls.",
		}, []string{"provider", "model", "status"}),
		CallLatencySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vciso_llm_call_latency_seconds",
			Help:    "Latency of provider calls.",
			Buckets: prometheus.DefBuckets,
		}, []string{"provider", "model"}),
		TokensTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_tokens_total",
			Help: "Tokens consumed by provider and token type.",
		}, []string{"provider", "model", "type"}),
		CostUSDTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_cost_usd_total",
			Help: "Estimated USD cost for vCISO LLM usage.",
		}, []string{"provider", "model", "tenant_id"}),
		ToolLoopIterations: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "vciso_llm_tool_loop_iterations",
			Help:    "Number of tool-loop iterations per LLM query.",
			Buckets: []float64{1, 2, 3, 4, 5},
		}),
		ToolCallsPerQuery: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "vciso_llm_tool_calls_per_query",
			Help:    "Number of tool calls per LLM query.",
			Buckets: []float64{0, 1, 2, 3, 4, 5, 10},
		}),
		GroundingResultsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_grounding_results_total",
			Help: "Grounding outcomes for LLM responses.",
		}, []string{"result"}),
		InjectionDetectionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_injection_detections_total",
			Help: "Prompt injection detections by severity.",
		}, []string{"severity"}),
		RateLimitRejectionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_rate_limit_rejections_total",
			Help: "LLM rate-limit rejections by tenant and limit type.",
		}, []string{"tenant_id", "limit_type"}),
		FallbackTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vciso_llm_fallback_total",
			Help: "Fallback executions by reason.",
		}, []string{"reason"}),
		ResponseLatencySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vciso_llm_response_latency_seconds",
			Help:    "Total user-visible response latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"engine"}),
		ContextTokensUsed: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "vciso_llm_context_tokens_used",
			Help:    "Estimated tokens used by compiled prompt context.",
			Buckets: []float64{512, 1024, 2048, 4096, 8192, 16384},
		}),
	}
	reg.MustRegister(
		m.QueriesTotal,
		m.CallsTotal,
		m.CallLatencySeconds,
		m.TokensTotal,
		m.CostUSDTotal,
		m.ToolLoopIterations,
		m.ToolCallsPerQuery,
		m.GroundingResultsTotal,
		m.InjectionDetectionsTotal,
		m.RateLimitRejectionsTotal,
		m.FallbackTotal,
		m.ResponseLatencySeconds,
		m.ContextTokensUsed,
	)
	return m
}
