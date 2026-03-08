package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	ContractsTotal           *prometheus.GaugeVec
	ContractsRiskDistribution *prometheus.GaugeVec
	ContractAnalysisDuration prometheus.Histogram
	ClauseExtractionTotal    *prometheus.CounterVec
	ClauseRiskTotal          *prometheus.CounterVec
	MissingClausesTotal      *prometheus.CounterVec
	ExpiringContracts        *prometheus.GaugeVec
	ComplianceAlertsTotal    *prometheus.GaugeVec
	ComplianceScore          *prometheus.GaugeVec
	DocumentsTotal           *prometheus.GaugeVec
	ContractValueTotal       *prometheus.GaugeVec
	WorkflowActive           prometheus.Gauge

	// Monitoring alert metrics (used by clario360-alerts.yaml)
	ContractsExpiring7d prometheus.Gauge // clario360_lex_contracts_expiring_7d
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ContractsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_contracts_total",
			Help: "Current number of contracts by tenant, type, and status.",
		}, []string{"tenant_id", "type", "status"}),
		ContractsRiskDistribution: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_contracts_risk_distribution",
			Help: "Current number of contracts by risk level.",
		}, []string{"risk_level"}),
		ContractAnalysisDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "lex_contract_analysis_duration_seconds",
			Help:    "Duration of deterministic contract analysis runs.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 3, 5, 8},
		}),
		ClauseExtractionTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lex_clause_extraction_total",
			Help: "Total number of clauses extracted by clause type.",
		}, []string{"clause_type"}),
		ClauseRiskTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lex_clause_risk_total",
			Help: "Total number of clause risk classifications by risk level.",
		}, []string{"risk_level"}),
		MissingClausesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lex_missing_clauses_total",
			Help: "Total missing standard clauses detected by clause type.",
		}, []string{"clause_type"}),
		ExpiringContracts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_expiring_contracts",
			Help: "Current number of active contracts in an expiry horizon.",
		}, []string{"horizon_days"}),
		ComplianceAlertsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_compliance_alerts_total",
			Help: "Current compliance alerts grouped by severity and status.",
		}, []string{"severity", "status"}),
		ComplianceScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_compliance_score",
			Help: "Calculated tenant compliance score.",
		}, []string{"tenant_id"}),
		DocumentsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_documents_total",
			Help: "Current legal documents by type.",
		}, []string{"type"}),
		ContractValueTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lex_contract_value_total",
			Help: "Current total contract value grouped by type and currency.",
		}, []string{"type", "currency"}),
		WorkflowActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lex_workflow_active",
			Help: "Current number of active legal workflow instances.",
		}),
		ContractsExpiring7d: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "clario360_lex_contracts_expiring_7d",
			Help: "Number of contracts expiring within 7 days without renewal action (used by Prometheus alerting rules).",
		}),
	}

	reg.MustRegister(
		m.ContractsTotal,
		m.ContractsRiskDistribution,
		m.ContractAnalysisDuration,
		m.ClauseExtractionTotal,
		m.ClauseRiskTotal,
		m.MissingClausesTotal,
		m.ExpiringContracts,
		m.ComplianceAlertsTotal,
		m.ComplianceScore,
		m.DocumentsTotal,
		m.ContractValueTotal,
		m.WorkflowActive,
		m.ContractsExpiring7d,
	)
	return m
}
