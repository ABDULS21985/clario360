package security

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for security events.
type Metrics struct {
	// CSRF
	CSRFFailures *prometheus.CounterVec

	// Input validation
	InjectionAttempts *prometheus.CounterVec
	XSSAttempts       *prometheus.CounterVec
	SanitizedInputs   prometheus.Counter

	// Rate limiting
	RateLimitHits      *prometheus.CounterVec
	AccountLockouts    prometheus.Counter
	EscalationTriggers prometheus.Counter

	// Auth security
	AuthFailures    *prometheus.CounterVec
	SessionEvents   *prometheus.CounterVec
	TokenOperations *prometheus.CounterVec

	// File upload
	FileUploadBlocked *prometheus.CounterVec
	FileUploadScanned prometheus.Counter

	// API security
	BOLAAttempts       prometheus.Counter
	BFLAAttempts       prometheus.Counter
	MassAssignment     prometheus.Counter
	SSRFBlocked        prometheus.Counter
	PathTraversal      prometheus.Counter
	InvalidContentType prometheus.Counter

	// Headers
	SecurityHeadersApplied prometheus.Counter
	CSPViolations          prometheus.Counter

	// General
	SecurityEventsTotal *prometheus.CounterVec
	BlockedRequests     *prometheus.CounterVec
}

// NewMetrics creates a new Metrics instance registered with the given registry.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		CSRFFailures: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "csrf_failures_total",
			Help:      "Total CSRF validation failures by reason.",
		}, []string{"reason"}),

		InjectionAttempts: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "injection_attempts_total",
			Help:      "Total SQL injection attempts detected by pattern category.",
		}, []string{"category"}),

		XSSAttempts: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "xss_attempts_total",
			Help:      "Total XSS attempts detected by pattern category.",
		}, []string{"category"}),

		SanitizedInputs: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "sanitized_inputs_total",
			Help:      "Total number of inputs that were sanitized.",
		}),

		RateLimitHits: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "rate_limit_hits_total",
			Help:      "Total rate limit hits by endpoint category.",
		}, []string{"category"}),

		AccountLockouts: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "account_lockouts_total",
			Help:      "Total account lockouts triggered.",
		}),

		EscalationTriggers: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "escalation_triggers_total",
			Help:      "Total security escalation events triggered.",
		}),

		AuthFailures: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "auth_failures_total",
			Help:      "Total authentication failures by reason.",
		}, []string{"reason"}),

		SessionEvents: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "session_events_total",
			Help:      "Total session security events by type.",
		}, []string{"event"}),

		TokenOperations: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "token_operations_total",
			Help:      "Total token operations by type.",
		}, []string{"operation"}),

		FileUploadBlocked: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "file_upload_blocked_total",
			Help:      "Total file uploads blocked by reason.",
		}, []string{"reason"}),

		FileUploadScanned: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "file_upload_scanned_total",
			Help:      "Total file uploads scanned for malware.",
		}),

		BOLAAttempts: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "bola_attempts_total",
			Help:      "Total Broken Object Level Authorization attempts detected.",
		}),

		BFLAAttempts: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "bfla_attempts_total",
			Help:      "Total Broken Function Level Authorization attempts detected.",
		}),

		MassAssignment: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "mass_assignment_attempts_total",
			Help:      "Total mass assignment attempts detected.",
		}),

		SSRFBlocked: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "ssrf_blocked_total",
			Help:      "Total SSRF attempts blocked.",
		}),

		PathTraversal: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "path_traversal_attempts_total",
			Help:      "Total path traversal attempts detected.",
		}),

		InvalidContentType: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "invalid_content_type_total",
			Help:      "Total requests blocked for invalid content type.",
		}),

		SecurityHeadersApplied: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "headers_applied_total",
			Help:      "Total responses with security headers applied.",
		}),

		CSPViolations: f.NewCounter(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "csp_violations_total",
			Help:      "Total CSP violations reported.",
		}),

		SecurityEventsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "events_total",
			Help:      "Total security events by severity and type.",
		}, []string{"severity", "type"}),

		BlockedRequests: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "security",
			Name:      "blocked_requests_total",
			Help:      "Total requests blocked by security controls.",
		}, []string{"control", "reason"}),
	}
}
