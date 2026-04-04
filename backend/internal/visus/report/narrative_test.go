package report

import (
	"strings"
	"testing"
	"time"
)

func TestNarrative_AllSections(t *testing.T) {
	narrative := GenerateNarrative(map[string]interface{}{
		"security_posture":  map[string]any{"available": true, "risk_score": 72.0, "grade": "B", "trend_word": "improved", "prev_risk_score": 80.0, "open_alerts": 10.0, "critical_alerts": 2.0, "mttr_hours": 4.2, "coverage": 61.0},
		"data_intelligence": map[string]any{"available": true, "quality_score": 91.0, "quality_grade": "A-", "success_rate": 95.0, "failed_count": 3.0, "contradiction_count": 5.0},
		"governance":        map[string]any{"available": true, "compliance_score": 86.0, "meeting_count": 2.0, "overdue_count": 4.0, "minutes_pending": 1.0},
		"legal":             map[string]any{"available": true, "active_contracts": 30.0, "value": 1500000.0, "expiring_count": 2.0, "high_risk_count": 1.0},
		"recommendations":   map[string]any{"items": []string{"Address the data quality backlog."}},
	}, [2]time.Time{time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)})

	if !strings.Contains(narrative, "Executive Summary") || !strings.Contains(narrative, "Address the data quality backlog.") {
		t.Fatalf("unexpected narrative output: %s", narrative)
	}
}

func TestNarrative_MissingSuite(t *testing.T) {
	narrative := GenerateNarrative(map[string]interface{}{
		"security_posture":  map[string]any{"available": true, "risk_score": 72.0, "grade": "B", "trend_word": "improved", "prev_risk_score": 80.0, "open_alerts": 10.0, "critical_alerts": 2.0, "mttr_hours": 4.2, "coverage": 61.0},
		"data_intelligence": map[string]any{"available": false},
		"governance":        map[string]any{"available": true, "compliance_score": 86.0, "meeting_count": 2.0, "overdue_count": 4.0, "minutes_pending": 1.0},
		"legal":             map[string]any{"available": true, "active_contracts": 30.0, "value": 1500000.0, "expiring_count": 2.0, "high_risk_count": 1.0},
		"recommendations":   map[string]any{"items": []string{"Stay the course."}},
	}, [2]time.Time{time.Now().UTC(), time.Now().UTC()})

	if !strings.Contains(narrative, "Data unavailable for this section.") {
		t.Fatalf("expected unavailable text, got %s", narrative)
	}
}

func TestNarrative_TrendWords(t *testing.T) {
	if got := TrendWord(3, true); got != "improved" {
		t.Fatalf("expected improved, got %s", got)
	}
	if got := TrendWord(-3, true); got != "declined" {
		t.Fatalf("expected declined, got %s", got)
	}
}

func TestNarrative_DirectionAware(t *testing.T) {
	if got := TrendWord(-4, false); got != "improved" {
		t.Fatalf("expected improved for lower-is-better decrease, got %s", got)
	}
}
