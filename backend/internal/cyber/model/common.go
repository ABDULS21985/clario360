package model

import "strings"

// Severity is the standard severity scale used across cyber entities.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ValidSeverities contains the allowed severity values.
var ValidSeverities = []Severity{
	SeverityCritical,
	SeverityHigh,
	SeverityMedium,
	SeverityLow,
	SeverityInfo,
}

// IsValid reports whether s is a recognized severity value.
func (s Severity) IsValid() bool {
	for _, candidate := range ValidSeverities {
		if candidate == s {
			return true
		}
	}
	return false
}

// Rank returns a sortable numeric rank where larger is more severe.
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// Normalize returns the lower-cased string form used in persistence and APIs.
func (s Severity) Normalize() Severity {
	return Severity(strings.ToLower(string(s)))
}

// AlertStatus captures the lifecycle state of an alert.
type AlertStatus string

const (
	AlertStatusNew           AlertStatus = "new"
	AlertStatusAcknowledged  AlertStatus = "acknowledged"
	AlertStatusInvestigating AlertStatus = "investigating"
	AlertStatusInProgress    AlertStatus = "in_progress"
	AlertStatusResolved      AlertStatus = "resolved"
	AlertStatusClosed        AlertStatus = "closed"
	AlertStatusFalsePositive AlertStatus = "false_positive"
	AlertStatusEscalated     AlertStatus = "escalated"
	AlertStatusMerged        AlertStatus = "merged"
)

// ValidAlertStatuses contains the allowed alert statuses.
var ValidAlertStatuses = []AlertStatus{
	AlertStatusNew,
	AlertStatusAcknowledged,
	AlertStatusInvestigating,
	AlertStatusInProgress,
	AlertStatusResolved,
	AlertStatusClosed,
	AlertStatusFalsePositive,
	AlertStatusEscalated,
	AlertStatusMerged,
}

// IsValid reports whether s is a recognized alert status.
func (s AlertStatus) IsValid() bool {
	for _, candidate := range ValidAlertStatuses {
		if candidate == s {
			return true
		}
	}
	return false
}

// IsOpen reports whether the alert should still participate in deduplication.
func (s AlertStatus) IsOpen() bool {
	switch s {
	case AlertStatusNew, AlertStatusAcknowledged, AlertStatusInvestigating, AlertStatusInProgress, AlertStatusEscalated:
		return true
	default:
		return false
	}
}

// DetectionRuleType captures the evaluator used to process a rule.
type DetectionRuleType string

const (
	RuleTypeSigma       DetectionRuleType = "sigma"
	RuleTypeThreshold   DetectionRuleType = "threshold"
	RuleTypeCorrelation DetectionRuleType = "correlation"
	RuleTypeAnomaly     DetectionRuleType = "anomaly"
)

// ValidDetectionRuleTypes contains all supported rule types.
var ValidDetectionRuleTypes = []DetectionRuleType{
	RuleTypeSigma,
	RuleTypeThreshold,
	RuleTypeCorrelation,
	RuleTypeAnomaly,
}

// IsValid reports whether t is a supported rule type.
func (t DetectionRuleType) IsValid() bool {
	for _, candidate := range ValidDetectionRuleTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// ThreatType captures the classification of a threat record.
type ThreatType string

const (
	ThreatTypeMalware     ThreatType = "malware"
	ThreatTypePhishing    ThreatType = "phishing"
	ThreatTypeAPT         ThreatType = "apt"
	ThreatTypeRansomware  ThreatType = "ransomware"
	ThreatTypeDDoS        ThreatType = "ddos"
	ThreatTypeInsider     ThreatType = "insider_threat"
	ThreatTypeSupplyChain ThreatType = "supply_chain"
	ThreatTypeZeroDay     ThreatType = "zero_day"
	ThreatTypeBruteForce  ThreatType = "brute_force"
	ThreatTypeOther       ThreatType = "other"
)

// ValidThreatTypes contains the allowed threat categories.
var ValidThreatTypes = []ThreatType{
	ThreatTypeMalware,
	ThreatTypePhishing,
	ThreatTypeAPT,
	ThreatTypeRansomware,
	ThreatTypeDDoS,
	ThreatTypeInsider,
	ThreatTypeSupplyChain,
	ThreatTypeZeroDay,
	ThreatTypeBruteForce,
	ThreatTypeOther,
}

// IsValid reports whether t is a supported threat type.
func (t ThreatType) IsValid() bool {
	for _, candidate := range ValidThreatTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// ThreatStatus captures the lifecycle of a tracked threat.
type ThreatStatus string

const (
	ThreatStatusActive     ThreatStatus = "active"
	ThreatStatusContained  ThreatStatus = "contained"
	ThreatStatusEradicated ThreatStatus = "eradicated"
	ThreatStatusMonitoring ThreatStatus = "monitoring"
	ThreatStatusClosed     ThreatStatus = "closed"
)

// ValidThreatStatuses contains the supported threat statuses.
var ValidThreatStatuses = []ThreatStatus{
	ThreatStatusActive,
	ThreatStatusContained,
	ThreatStatusEradicated,
	ThreatStatusMonitoring,
	ThreatStatusClosed,
}

// IsValid reports whether s is a supported threat status.
func (s ThreatStatus) IsValid() bool {
	for _, candidate := range ValidThreatStatuses {
		if candidate == s {
			return true
		}
	}
	return false
}

// IndicatorType captures the IOC type.
type IndicatorType string

const (
	IndicatorTypeIP          IndicatorType = "ip"
	IndicatorTypeDomain      IndicatorType = "domain"
	IndicatorTypeURL         IndicatorType = "url"
	IndicatorTypeEmail       IndicatorType = "email"
	IndicatorTypeHashMD5     IndicatorType = "file_hash_md5"
	IndicatorTypeHashSHA1    IndicatorType = "file_hash_sha1"
	IndicatorTypeHashSHA256  IndicatorType = "file_hash_sha256"
	IndicatorTypeCertificate IndicatorType = "certificate"
	IndicatorTypeRegistryKey IndicatorType = "registry_key"
	IndicatorTypeUserAgent   IndicatorType = "user_agent"
	IndicatorTypeCIDR        IndicatorType = "cidr"
)

// ValidIndicatorTypes contains the supported IOC types.
var ValidIndicatorTypes = []IndicatorType{
	IndicatorTypeIP,
	IndicatorTypeDomain,
	IndicatorTypeURL,
	IndicatorTypeEmail,
	IndicatorTypeHashMD5,
	IndicatorTypeHashSHA1,
	IndicatorTypeHashSHA256,
	IndicatorTypeCertificate,
	IndicatorTypeRegistryKey,
	IndicatorTypeUserAgent,
	IndicatorTypeCIDR,
}

// IsValid reports whether t is a supported indicator type.
func (t IndicatorType) IsValid() bool {
	for _, candidate := range ValidIndicatorTypes {
		if candidate == t {
			return true
		}
	}
	return false
}
