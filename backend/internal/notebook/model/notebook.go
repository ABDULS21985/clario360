package model

import (
	"errors"
	"time"
)

var (
	ErrInvalidProfile   = errors.New("invalid notebook profile")
	ErrProfileForbidden = errors.New("profile is not permitted for this user")
	ErrServerRunning    = errors.New("notebook server already running")
	ErrServerNotFound   = errors.New("notebook server not found")
	ErrTemplateNotFound = errors.New("notebook template not found")
	ErrActivityInvalid  = errors.New("notebook activity payload is invalid")
)

type Actor struct {
	UserID   string
	TenantID string
	Email    string
	Roles    []string
}

type NotebookServer struct {
	ID            string     `json:"id"`
	Profile       string     `json:"profile"`
	Status        string     `json:"status"`
	URL           string     `json:"url"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	LastActivity  *time.Time `json:"last_activity,omitempty"`
	CPUPercent    float64    `json:"cpu_percent"`
	MemoryMB      int        `json:"memory_mb"`
	MemoryLimitMB int        `json:"memory_limit_mb"`
}

type NotebookServerStatus struct {
	ID            string     `json:"id"`
	Profile       string     `json:"profile"`
	Status        string     `json:"status"`
	CPUPercent    float64    `json:"cpu_percent"`
	MemoryMB      int        `json:"memory_mb"`
	MemoryLimitMB int        `json:"memory_limit_mb"`
	UptimeSeconds int64      `json:"uptime_seconds"`
	LastActivity  *time.Time `json:"last_activity,omitempty"`
}

type NotebookProfile struct {
	Slug         string   `json:"slug"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	CPU           string   `json:"cpu"`
	Memory       string   `json:"memory"`
	Storage      string   `json:"storage"`
	RequiresRole []string `json:"requires_role,omitempty"`
	SparkEnabled bool     `json:"spark_enabled"`
	Default      bool     `json:"default"`
}

type NotebookTemplate struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Difficulty  string   `json:"difficulty"`
	Tags        []string `json:"tags"`
	Filename    string   `json:"filename"`
}

type CopiedTemplate struct {
	TemplateID string `json:"template_id"`
	Path       string `json:"path"`
	OpenURL    string `json:"open_url"`
}

type ActivityKind string

const (
	ActivitySDKCall   ActivityKind = "sdk_api"
	ActivityDataQuery ActivityKind = "data_query"
	ActivitySparkJob  ActivityKind = "spark_job"
)

type ActivityRecord struct {
	Kind        ActivityKind       `json:"kind"`
	Endpoint    string             `json:"endpoint,omitempty"`
	Status      string             `json:"status,omitempty"`
	Source      string             `json:"source,omitempty"`
	Description string             `json:"description,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
	OccurredAt  time.Time          `json:"occurred_at"`
}

func DefaultProfiles() []NotebookProfile {
	return []NotebookProfile{
		{
			Slug:        "soc-analyst",
			DisplayName: "SOC Analyst",
			Description: "Security analysis, incident investigation, threat hunting, and reporting.",
			CPU:         "2 CPU",
			Memory:      "4 GB",
			Storage:     "5 GiB",
			Default:     true,
		},
		{
			Slug:        "data-scientist",
			DisplayName: "Data Scientist",
			Description: "Model development, evaluation, and feature engineering for governed AI workloads.",
			CPU:         "4 CPU",
			Memory:      "8 GB",
			Storage:     "20 GiB",
		},
		{
			Slug:         "spark-connected",
			DisplayName:  "Spark Connected",
			Description:  "Large-scale analytics with PySpark, HDFS access, and cluster-attached jobs.",
			CPU:          "8 CPU",
			Memory:       "16 GB",
			Storage:      "50 GiB",
			SparkEnabled: true,
		},
		{
			Slug:         "admin",
			DisplayName:  "Admin",
			Description:  "Full-access profile for tenant administrators and security managers.",
			CPU:          "8 CPU",
			Memory:       "32 GB",
			Storage:      "100 GiB",
			RequiresRole: []string{"tenant-admin", "security-manager", "super-admin"},
		},
	}
}

func DefaultTemplates() []NotebookTemplate {
	return []NotebookTemplate{
		{ID: "01_threat_detection_quickstart", Title: "Threat Detection Quickstart", Description: "Pull recent alerts, visualize trends, and export a critical-alert report.", Difficulty: "beginner", Tags: []string{"security", "alerts", "visualization"}, Filename: "01_threat_detection_quickstart.ipynb"},
		{ID: "02_anomaly_detection_tutorial", Title: "Anomaly Detection Tutorial", Description: "Build statistical baselines over event streams and flag deviations.", Difficulty: "intermediate", Tags: []string{"security", "anomaly", "statistics"}, Filename: "02_anomaly_detection_tutorial.ipynb"},
		{ID: "03_ueba_behavioral_analysis", Title: "UEBA Behavioral Analysis", Description: "Model user behavior from data-access telemetry and raise anomaly candidates.", Difficulty: "advanced", Tags: []string{"ueba", "behavior", "clickhouse"}, Filename: "03_ueba_behavioral_analysis.ipynb"},
		{ID: "04_custom_detection_rule_builder", Title: "Custom Detection Rule Builder", Description: "Author, test, and deploy Sigma-like detection rules with governed rollout.", Difficulty: "intermediate", Tags: []string{"rules", "detections", "sigma"}, Filename: "04_custom_detection_rule_builder.ipynb"},
		{ID: "05_model_validation_framework", Title: "Model Validation Framework", Description: "Evaluate model precision, recall, ROC, and shadow-promotion readiness.", Difficulty: "advanced", Tags: []string{"ai", "governance", "validation"}, Filename: "05_model_validation_framework.ipynb"},
		{ID: "06_data_access_audit_analysis", Title: "Data Access Audit Analysis", Description: "Investigate access patterns for compliance, DSPM, and insider-risk review.", Difficulty: "intermediate", Tags: []string{"audit", "data", "compliance"}, Filename: "06_data_access_audit_analysis.ipynb"},
		{ID: "07_incident_investigation_playbook", Title: "Incident Investigation Playbook", Description: "Run a guided investigation from alert context to containment summary.", Difficulty: "beginner", Tags: []string{"incident-response", "mitre", "assets"}, Filename: "07_incident_investigation_playbook.ipynb"},
		{ID: "08_spark_large_scale_analysis", Title: "Spark Large-Scale Analysis", Description: "Correlate 30 days of security telemetry with Spark and ClickHouse.", Difficulty: "advanced", Tags: []string{"spark", "big-data", "correlation"}, Filename: "08_spark_large_scale_analysis.ipynb"},
		{ID: "09_threat_hunting_with_mitre", Title: "Threat Hunting with MITRE ATT&CK", Description: "Use ATT&CK coverage gaps to drive repeatable hunt hypotheses and findings.", Difficulty: "advanced", Tags: []string{"mitre", "threat-hunting", "coverage"}, Filename: "09_threat_hunting_with_mitre.ipynb"},
		{ID: "10_model_deployment_pipeline", Title: "Model Deployment Pipeline", Description: "Train, validate, shadow, compare, and promote a governed AI model version.", Difficulty: "advanced", Tags: []string{"ai", "shadow-mode", "promotion"}, Filename: "10_model_deployment_pipeline.ipynb"},
	}
}
