package metrics

import "github.com/prometheus/client_golang/prometheus"

// Metrics holds all Prometheus metrics for the IAM service.
type Metrics struct {
	// Authentication metrics
	LoginAttemptsTotal  *prometheus.CounterVec
	LoginSuccessTotal   *prometheus.CounterVec
	LoginDuration       *prometheus.HistogramVec
	TokensIssuedTotal   *prometheus.CounterVec
	TokenRefreshTotal   *prometheus.CounterVec
	MFAVerifyTotal      *prometheus.CounterVec
	PasswordResetTotal  *prometheus.CounterVec
	SessionsActive      *prometheus.GaugeVec
	APIKeysActive       *prometheus.GaugeVec
	UsersTotal          *prometheus.GaugeVec
	RolesTotal          *prometheus.GaugeVec

	// Monitoring alert metrics (used by clario360-alerts.yaml)
	LoginFailuresTotal *prometheus.CounterVec // clario360_auth_login_failures_total{ip}
}

// New creates a new Metrics instance and registers all collectors.
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	m := &Metrics{
		LoginAttemptsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_login_attempts_total",
			Help: "Total login attempts by method and result.",
		}, []string{"method", "result"}),
		LoginSuccessTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_login_success_total",
			Help: "Total successful logins by method.",
		}, []string{"method"}),
		LoginDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "iam_login_duration_seconds",
			Help:    "Login request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		TokensIssuedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_tokens_issued_total",
			Help: "Total tokens issued by type (access, refresh).",
		}, []string{"type"}),
		TokenRefreshTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_token_refresh_total",
			Help: "Total token refresh attempts by result.",
		}, []string{"result"}),
		MFAVerifyTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_mfa_verify_total",
			Help: "Total MFA verification attempts by method and result.",
		}, []string{"method", "result"}),
		PasswordResetTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "iam_password_reset_total",
			Help: "Total password reset requests by result.",
		}, []string{"result"}),
		SessionsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "iam_sessions_active",
			Help: "Current active sessions by tenant.",
		}, []string{"tenant_id"}),
		APIKeysActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "iam_api_keys_active",
			Help: "Current active API keys by tenant.",
		}, []string{"tenant_id"}),
		UsersTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "iam_users_total",
			Help: "Current user count by tenant and status.",
		}, []string{"tenant_id", "status"}),
		RolesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "iam_roles_total",
			Help: "Current role count by tenant.",
		}, []string{"tenant_id"}),
		LoginFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "clario360_auth_login_failures_total",
			Help: "Total failed login attempts by IP (used by Prometheus alerting rules for brute force detection).",
		}, []string{"ip"}),
	}

	reg.MustRegister(
		m.LoginAttemptsTotal,
		m.LoginSuccessTotal,
		m.LoginDuration,
		m.TokensIssuedTotal,
		m.TokenRefreshTotal,
		m.MFAVerifyTotal,
		m.PasswordResetTotal,
		m.SessionsActive,
		m.APIKeysActive,
		m.UsersTotal,
		m.RolesTotal,
		m.LoginFailuresTotal,
	)

	return m
}
