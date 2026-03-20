package service

import (
	"github.com/prometheus/client_golang/prometheus"

	obsmetrics "github.com/clario360/platform/internal/observability/metrics"
)

type Metrics struct {
	registrationsTotal         *prometheus.CounterVec
	otpVerificationsTotal      *prometheus.CounterVec
	provisioningTotal          *prometheus.CounterVec
	provisioningDuration       *prometheus.HistogramVec
	provisioningStepDuration   *prometheus.HistogramVec
	wizardCompletionsTotal     *prometheus.CounterVec
	wizardStepCompletionsTotal *prometheus.CounterVec
	wizardAbandonmentTotal     *prometheus.CounterVec
	invitationsTotal           *prometheus.CounterVec
	invitationAcceptanceRate   *prometheus.GaugeVec
	deprovisionsTotal          *prometheus.CounterVec
	timeToActive               *prometheus.HistogramVec
}

func NewMetrics(registry *obsmetrics.Metrics) *Metrics {
	if registry == nil {
		return &Metrics{}
	}
	return &Metrics{
		registrationsTotal: registry.Counter(
			"onboarding_registrations_total",
			"Total onboarding registration attempts.",
			[]string{"status"},
		),
		otpVerificationsTotal: registry.Counter(
			"onboarding_otp_verifications_total",
			"OTP verification attempts by result.",
			[]string{"result"},
		),
		provisioningTotal: registry.Counter(
			"onboarding_provisioning_total",
			"Tenant provisioning runs by status.",
			[]string{"status"},
		),
		provisioningDuration: registry.Histogram(
			"onboarding_provisioning_duration_seconds",
			"End-to-end provisioning duration in seconds.",
			[]string{"status"},
			prometheus.DefBuckets,
		),
		provisioningStepDuration: registry.Histogram(
			"onboarding_provisioning_step_duration_seconds",
			"Provisioning step duration in seconds.",
			[]string{"step_name"},
			prometheus.DefBuckets,
		),
		wizardCompletionsTotal: registry.Counter(
			"onboarding_wizard_completions_total",
			"Completed onboarding wizard sessions.",
			nil,
		),
		wizardStepCompletionsTotal: registry.Counter(
			"onboarding_wizard_step_completions_total",
			"Completed onboarding wizard steps.",
			[]string{"step"},
		),
		wizardAbandonmentTotal: registry.Counter(
			"onboarding_wizard_abandonment_total",
			"Wizard abandonment count by last step.",
			[]string{"last_step"},
		),
		invitationsTotal: registry.Counter(
			"onboarding_invitations_total",
			"Invitation lifecycle totals.",
			[]string{"status"},
		),
		invitationAcceptanceRate: registry.Gauge(
			"onboarding_invitation_acceptance_rate",
			"Approximate invitation acceptance rate per tenant.",
			[]string{"tenant_id"},
		),
		deprovisionsTotal: registry.Counter(
			"onboarding_deprovisions_total",
			"Total tenant deprovision operations.",
			nil,
		),
		timeToActive: registry.Histogram(
			"onboarding_tenant_time_to_active_seconds",
			"Elapsed time from registration to tenant activation.",
			[]string{"tenant_id"},
			prometheus.DefBuckets,
		),
	}
}
