package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WorkflowInstancesTotal tracks the total number of workflow instances
	// created, labelled by the definition name and terminal status.
	WorkflowInstancesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "instances_total",
		Help:      "Total workflow instances created by definition and status.",
	}, []string{"definition_name", "status"})

	// WorkflowStepDuration tracks the execution duration of individual workflow
	// steps in seconds, labelled by step type and definition name.
	WorkflowStepDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "workflow",
		Name:      "step_duration_seconds",
		Help:      "Duration of workflow step executions in seconds.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"step_type", "definition_name"})

	// WorkflowTasksTotal tracks the total number of human tasks by status and
	// definition name.
	WorkflowTasksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "tasks_total",
		Help:      "Total human tasks by status and definition.",
	}, []string{"status", "definition_name"})

	// WorkflowSLABreachesTotal tracks SLA breaches on human tasks.
	WorkflowSLABreachesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "sla_breaches_total",
		Help:      "Total SLA breaches by definition and step.",
	}, []string{"definition_name", "step_id"})

	// WorkflowActiveInstances tracks the current number of running (non-terminal)
	// workflow instances per definition.
	WorkflowActiveInstances = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "workflow",
		Name:      "active_instances",
		Help:      "Current number of active (running) workflow instances by definition.",
	}, []string{"definition_name"})

	// WorkflowTimersFired counts the total number of timer steps that have fired.
	WorkflowTimersFired = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "timers_fired_total",
		Help:      "Total timer steps that have fired.",
	})

	// WorkflowEngineErrors tracks internal engine errors by operation.
	WorkflowEngineErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "engine_errors_total",
		Help:      "Total internal engine errors by operation.",
	}, []string{"operation"})

	// WorkflowServiceTaskRetries tracks service task retry attempts.
	WorkflowServiceTaskRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflow",
		Name:      "service_task_retries_total",
		Help:      "Total service task retry attempts by definition and step.",
	}, []string{"definition_name", "step_id"})
)

// RecordInstanceStarted increments the instance counter for the running status.
func RecordInstanceStarted(definitionName string) {
	WorkflowInstancesTotal.WithLabelValues(definitionName, "running").Inc()
	WorkflowActiveInstances.WithLabelValues(definitionName).Inc()
}

// RecordInstanceCompleted increments the instance counter for completion and
// decrements the active gauge.
func RecordInstanceCompleted(definitionName, status string) {
	WorkflowInstancesTotal.WithLabelValues(definitionName, status).Inc()
	WorkflowActiveInstances.WithLabelValues(definitionName).Dec()
}

// RecordStepDuration records the duration of a step execution in seconds.
func RecordStepDuration(stepType, definitionName string, durationSeconds float64) {
	WorkflowStepDuration.WithLabelValues(stepType, definitionName).Observe(durationSeconds)
}

// RecordTaskCreated increments the task counter for the pending status.
func RecordTaskCreated(definitionName string) {
	WorkflowTasksTotal.WithLabelValues("pending", definitionName).Inc()
}

// RecordTaskStatusChange increments the task counter for the given status.
func RecordTaskStatusChange(status, definitionName string) {
	WorkflowTasksTotal.WithLabelValues(status, definitionName).Inc()
}

// RecordSLABreach increments the SLA breach counter.
func RecordSLABreach(definitionName, stepID string) {
	WorkflowSLABreachesTotal.WithLabelValues(definitionName, stepID).Inc()
}

// RecordTimerFired increments the timers fired counter.
func RecordTimerFired() {
	WorkflowTimersFired.Inc()
}
