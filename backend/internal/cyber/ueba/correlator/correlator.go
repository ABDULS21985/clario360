package correlator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type AnomalyCorrelator struct {
	window time.Duration
}

func New(window time.Duration) *AnomalyCorrelator {
	return &AnomalyCorrelator{window: window}
}

func (c *AnomalyCorrelator) Correlate(_ context.Context, tenantID uuid.UUID, entityID string, signals []model.AnomalySignal) []model.UEBAAlert {
	windowed := filterWithinWindow(signals, time.Now().UTC().Add(-c.window))
	if len(windowed) == 0 {
		return nil
	}

	matches := make([]ruleMatch, 0, 4)
	typeBuckets := bucketByType(windowed)

	if match := c.matchDataExfiltration(typeBuckets); match != nil {
		matches = append(matches, *match)
	}
	if match := c.matchCredentialCompromise(typeBuckets); match != nil {
		matches = append(matches, *match)
	}
	if match := c.matchInsiderThreat(typeBuckets); match != nil {
		matches = append(matches, *match)
	}
	if match := c.matchLateralMovement(typeBuckets); match != nil {
		matches = append(matches, *match)
	}
	if match := c.matchReconnaissance(typeBuckets); match != nil {
		matches = append(matches, *match)
	}
	if len(matches) == 0 && len(windowed) >= 3 {
		matches = append(matches, ruleMatch{
			alertType: model.AlertTypeUnusualActivity,
			signals:   windowed,
			severity:  severityFromSignals(windowed, 0, "low"),
		})
	}

	consumed := make(map[uuid.UUID]struct{}, len(windowed))
	alerts := make([]model.UEBAAlert, 0, len(matches)+len(windowed))
	for _, match := range matches {
		alert := buildAlert(tenantID, entityID, match.alertType, match.severity, match.signals, match.mitreTactic)
		alerts = append(alerts, alert)
		for _, signal := range match.signals {
			consumed[signal.EventID] = struct{}{}
		}
	}

	for _, signal := range windowed {
		if severityRank(signal.Severity) < severityRank("high") {
			continue
		}
		if _, ok := consumed[signal.EventID]; ok {
			continue
		}
		alerts = append(alerts, buildAlert(tenantID, entityID, model.AlertTypeUnusualActivity, signal.Severity, []model.AnomalySignal{signal}, signal.MITRETactic))
	}

	return alerts
}

func (c *AnomalyCorrelator) matchDataExfiltration(buckets map[model.SignalType][]model.AnomalySignal) *ruleMatch {
	if timeSignals, ok := buckets[model.SignalTypeUnusualTime]; ok {
		if volumeSignals, ok := buckets[model.SignalTypeUnusualVolume]; ok && len(volumeSignals) > 0 {
			signals := []model.AnomalySignal{timeSignals[0], volumeSignals[0]}
			return &ruleMatch{
				alertType:   model.AlertTypePossibleDataExfiltration,
				signals:     signals,
				severity:    severityFromSignals(signals, 1, "medium"),
				mitreTactic: "TA0010",
			}
		}
		if bulkSignals, ok := buckets[model.SignalTypeBulkDataAccess]; ok && len(bulkSignals) > 0 {
			signals := []model.AnomalySignal{timeSignals[0], bulkSignals[0]}
			return &ruleMatch{
				alertType:   model.AlertTypePossibleDataExfiltration,
				signals:     signals,
				severity:    severityFromSignals(signals, 1, "medium"),
				mitreTactic: "TA0010",
			}
		}
	}
	for _, signal := range buckets[model.SignalTypeUnusualVolume] {
		if signal.DeviationZ > 5 {
			return &ruleMatch{
				alertType:   model.AlertTypePossibleDataExfiltration,
				signals:     []model.AnomalySignal{signal},
				severity:    escalateSeverity(signal.Severity, 1),
				mitreTactic: "TA0010",
			}
		}
	}
	return nil
}

func (c *AnomalyCorrelator) matchCredentialCompromise(buckets map[model.SignalType][]model.AnomalySignal) *ruleMatch {
	ipSignals := buckets[model.SignalTypeNewSourceIP]
	if len(ipSignals) == 0 {
		return nil
	}
	if tableSignals := buckets[model.SignalTypeNewTableAccess]; len(tableSignals) > 0 {
		signals := []model.AnomalySignal{ipSignals[0], tableSignals[0]}
		return &ruleMatch{
			alertType:   model.AlertTypePossibleCredentialCompromise,
			signals:     signals,
			severity:    severityFromSignals(signals, 1, "medium"),
			mitreTactic: "TA0006",
		}
	}
	if timeSignals := buckets[model.SignalTypeUnusualTime]; len(timeSignals) > 0 {
		for signalType, otherSignals := range buckets {
			if signalType == model.SignalTypeNewSourceIP || signalType == model.SignalTypeUnusualTime || len(otherSignals) == 0 {
				continue
			}
			signals := []model.AnomalySignal{ipSignals[0], timeSignals[0], otherSignals[0]}
			return &ruleMatch{
				alertType:   model.AlertTypePossibleCredentialCompromise,
				signals:     signals,
				severity:    severityFromSignals(signals, 1, "medium"),
				mitreTactic: "TA0006",
			}
		}
	}
	return nil
}

func (c *AnomalyCorrelator) matchInsiderThreat(buckets map[model.SignalType][]model.AnomalySignal) *ruleMatch {
	privSignals := buckets[model.SignalTypePrivilegeEscalation]
	if len(privSignals) == 0 {
		return nil
	}
	if bulkSignals := buckets[model.SignalTypeBulkDataAccess]; len(bulkSignals) > 0 {
		signals := []model.AnomalySignal{privSignals[0], bulkSignals[0]}
		return &ruleMatch{
			alertType:   model.AlertTypePossibleInsiderThreat,
			signals:     signals,
			severity:    severityFromSignals(signals, 0, "high"),
			mitreTactic: "TA0004",
		}
	}
	for _, tableSignal := range buckets[model.SignalTypeNewTableAccess] {
		if strings.EqualFold(tableSignal.TableSensitivity, "restricted") {
			signals := []model.AnomalySignal{privSignals[0], tableSignal}
			return &ruleMatch{
				alertType:   model.AlertTypePossibleInsiderThreat,
				signals:     signals,
				severity:    severityFromSignals(signals, 0, "high"),
				mitreTactic: "TA0010",
			}
		}
	}
	return nil
}

func (c *AnomalyCorrelator) matchLateralMovement(buckets map[model.SignalType][]model.AnomalySignal) *ruleMatch {
	failureSignals := buckets[model.SignalTypeFailedAccessSpike]
	if len(failureSignals) == 0 {
		return nil
	}
	if ipSignals := buckets[model.SignalTypeNewSourceIP]; len(ipSignals) > 0 {
		signals := []model.AnomalySignal{failureSignals[0], ipSignals[0]}
		return &ruleMatch{
			alertType:   model.AlertTypePossibleLateralMovement,
			signals:     signals,
			severity:    "high",
			mitreTactic: "TA0008",
		}
	}
	if tableSignals := buckets[model.SignalTypeNewTableAccess]; len(tableSignals) > 0 {
		signals := []model.AnomalySignal{failureSignals[0], tableSignals[0]}
		return &ruleMatch{
			alertType:   model.AlertTypePossibleLateralMovement,
			signals:     signals,
			severity:    "high",
			mitreTactic: "TA0008",
		}
	}
	return nil
}

func (c *AnomalyCorrelator) matchReconnaissance(buckets map[model.SignalType][]model.AnomalySignal) *ruleMatch {
	tableSignals := buckets[model.SignalTypeNewTableAccess]
	if len(tableSignals) < 3 {
		return nil
	}
	seen := make(map[string]struct{}, len(tableSignals))
	selected := make([]model.AnomalySignal, 0, 3)
	for _, signal := range tableSignals {
		if _, ok := seen[signal.ActualValue]; ok {
			continue
		}
		seen[signal.ActualValue] = struct{}{}
		selected = append(selected, signal)
	}
	if len(selected) < 3 {
		return nil
	}
	return &ruleMatch{
		alertType:   model.AlertTypeDataReconnaissance,
		signals:     selected[:3],
		severity:    "medium",
		mitreTactic: "TA0007",
	}
}

func bucketByType(signals []model.AnomalySignal) map[model.SignalType][]model.AnomalySignal {
	buckets := make(map[model.SignalType][]model.AnomalySignal, len(signals))
	for _, signal := range signals {
		buckets[signal.SignalType] = append(buckets[signal.SignalType], signal)
	}
	return buckets
}

func buildAlert(tenantID uuid.UUID, entityID string, alertType model.AlertType, severity string, signals []model.AnomalySignal, tactic string) model.UEBAAlert {
	if len(signals) == 0 {
		return model.UEBAAlert{}
	}
	start := signals[0].EventTimestamp
	end := signals[len(signals)-1].EventTimestamp
	eventIDs := make([]uuid.UUID, 0, len(signals))
	techniques := make([]string, 0, len(signals))
	confidenceSum := 0.0
	expected := make([]map[string]any, 0, len(signals))
	actual := make([]map[string]any, 0, len(signals))
	for _, signal := range signals {
		eventIDs = append(eventIDs, signal.EventID)
		if signal.MITRETechnique != "" {
			techniques = append(techniques, signal.MITRETechnique)
		}
		confidenceSum += signal.Confidence
		expected = append(expected, map[string]any{
			"signal_type": signal.SignalType,
			"expected":    signal.ExpectedValue,
		})
		actual = append(actual, map[string]any{
			"signal_type": signal.SignalType,
			"actual":      signal.ActualValue,
		})
	}
	baseConfidence := confidenceSum / float64(len(signals))
	boost := minFloat(0.15*float64(len(signals)-1), 0.3)
	title := humanAlertTitle(alertType)
	description := fmt.Sprintf("%s triggered by %d correlated UEBA signals for entity %s.", title, len(signals), entityID)
	return model.UEBAAlert{
		TenantID:               tenantID,
		EntityID:               entityID,
		AlertType:              alertType,
		Severity:               severity,
		Confidence:             minFloat(baseConfidence+boost, 0.99),
		Title:                  title,
		Description:            description,
		TriggeringSignals:      signals,
		TriggeringEventIDs:     dedupUUIDs(eventIDs),
		BaselineComparison:     map[string]any{"expected": expected, "actual": actual},
		CorrelatedSignalCount:  len(signals),
		CorrelationWindowStart: start,
		CorrelationWindowEnd:   end,
		MITRETechniqueIDs:      dedupStrings(techniques),
		MITRETactic:            tactic,
		Status:                 "new",
	}
}

func humanAlertTitle(alertType model.AlertType) string {
	switch alertType {
	case model.AlertTypePossibleDataExfiltration:
		return "Possible data exfiltration"
	case model.AlertTypePossibleCredentialCompromise:
		return "Possible credential compromise"
	case model.AlertTypePossibleInsiderThreat:
		return "Possible insider threat"
	case model.AlertTypePossibleLateralMovement:
		return "Possible lateral movement"
	case model.AlertTypePossiblePrivilegeAbuse:
		return "Possible privilege abuse"
	case model.AlertTypeDataReconnaissance:
		return "Data reconnaissance"
	case model.AlertTypePolicyViolation:
		return "Policy violation"
	default:
		return "Unusual activity"
	}
}

func dedupUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dedupStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
