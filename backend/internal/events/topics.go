package events

// Topics holds all Kafka topic names for the Clario 360 platform.
// Naming convention: {domain}.{entity}.events
var Topics = struct {
	// Platform
	IAMEvents          string
	AuditEvents        string
	NotificationEvents string
	WorkflowEvents     string
	FileEvents         string

	// Cybersecurity
	AssetEvents       string
	VulnerabilityEvents string
	ThreatEvents      string
	AlertEvents       string
	RuleEvents        string
	CtemEvents        string
	RiskEvents        string
	RemediationEvents string

	// Data
	DataSourceEvents    string
	PipelineEvents      string
	QualityEvents       string
	ContradictionEvents string
	LineageEvents       string

	// Enterprise
	ActaEvents  string
	LexEvents   string
	VisusEvents string

	// Dead Letter
	DeadLetter string
}{
	// Platform
	IAMEvents:          "platform.iam.events",
	AuditEvents:        "platform.audit.events",
	NotificationEvents: "platform.notification.events",
	WorkflowEvents:     "platform.workflow.events",
	FileEvents:         "platform.file.events",

	// Cybersecurity
	AssetEvents:         "cyber.asset.events",
	VulnerabilityEvents: "cyber.vulnerability.events",
	ThreatEvents:        "cyber.threat.events",
	AlertEvents:         "cyber.alert.events",
	RuleEvents:          "cyber.rule.events",
	CtemEvents:          "cyber.ctem.events",
	RiskEvents:          "cyber.risk.events",
	RemediationEvents:   "cyber.remediation.events",

	// Data
	DataSourceEvents:    "data.source.events",
	PipelineEvents:      "data.pipeline.events",
	QualityEvents:       "data.quality.events",
	ContradictionEvents: "data.contradiction.events",
	LineageEvents:       "data.lineage.events",

	// Enterprise
	ActaEvents:  "enterprise.acta.events",
	LexEvents:   "enterprise.lex.events",
	VisusEvents: "enterprise.visus.events",

	// Dead Letter
	DeadLetter: "platform.dead-letter",
}

// Legacy topic constants for backward compatibility with existing service code.
const (
	TopicAuditLog      = "platform.audit.events"
	TopicUserCreated   = "platform.iam.events"
	TopicUserUpdated   = "platform.iam.events"
	TopicUserDeleted   = "platform.iam.events"
	TopicTenantCreated = "platform.iam.events"
	TopicWorkflowStart = "platform.workflow.events"
	TopicWorkflowEnd   = "platform.workflow.events"
	TopicCyberAlert    = "cyber.alert.events"
	TopicCyberRule     = "cyber.rule.events"
	TopicDataPipeline  = "data.pipeline.events"
	TopicActaDocument  = "enterprise.acta.events"
	TopicLexCase       = "enterprise.lex.events"
	TopicVisusReport   = "enterprise.visus.events"
)

// AllTopics returns all topic names for admin operations (e.g., topic creation).
func AllTopics() []string {
	return []string{
		Topics.IAMEvents,
		Topics.AuditEvents,
		Topics.NotificationEvents,
		Topics.WorkflowEvents,
		Topics.FileEvents,
		Topics.AssetEvents,
		Topics.VulnerabilityEvents,
		Topics.ThreatEvents,
		Topics.AlertEvents,
		Topics.RuleEvents,
		Topics.CtemEvents,
		Topics.RiskEvents,
		Topics.RemediationEvents,
		Topics.DataSourceEvents,
		Topics.PipelineEvents,
		Topics.QualityEvents,
		Topics.ContradictionEvents,
		Topics.LineageEvents,
		Topics.ActaEvents,
		Topics.LexEvents,
		Topics.VisusEvents,
		Topics.DeadLetter,
	}
}

// TopicConfig holds per-topic Kafka configuration for admin operations.
type TopicConfig struct {
	Name              string
	NumPartitions     int32
	ReplicationFactor int16
	RetentionMs       int64 // -1 for infinite
}

// DefaultTopicConfigs returns recommended configurations for all topics.
func DefaultTopicConfigs() []TopicConfig {
	configs := make([]TopicConfig, 0, len(AllTopics()))
	for _, topic := range AllTopics() {
		cfg := TopicConfig{
			Name:              topic,
			NumPartitions:     6,
			ReplicationFactor: 3,
			RetentionMs:       7 * 24 * 60 * 60 * 1000, // 7 days
		}

		switch topic {
		case Topics.DeadLetter:
			cfg.RetentionMs = 30 * 24 * 60 * 60 * 1000 // 30 days
			cfg.NumPartitions = 3
		case Topics.AuditEvents:
			cfg.RetentionMs = 90 * 24 * 60 * 60 * 1000 // 90 days
		}

		configs = append(configs, cfg)
	}
	return configs
}
