from __future__ import annotations

from clario360.models.cyber import Alert
from clario360.models.risk import RiskScore


def test_alert_from_dict() -> None:
    alert = Alert.from_dict(
        {
            "id": "alert-1",
            "title": "Suspicious login",
            "severity": "high",
            "status": "new",
            "confidence_score": 0.87,
            "created_at": "2026-03-09T00:00:00Z",
            "explanation": {
                "summary": "Impossible travel detected.",
                "recommended_actions": ["Reset the password", "Review MFA posture"],
            },
        }
    )
    assert alert.explanation is not None
    assert alert.explanation.summary == "Impossible travel detected."


def test_alert_to_dict_excludes_none() -> None:
    alert = Alert(
        id="alert-1",
        title="Suspicious login",
        severity="high",
        status="new",
        created_at="2026-03-09T00:00:00Z",
    )
    payload = alert.to_dict()
    assert "description" not in payload
    assert payload["title"] == "Suspicious login"


def test_risk_score_from_dict() -> None:
    risk = RiskScore.from_dict({"overall_score": 62, "grade": "D", "severity": "high"})
    assert risk.overall_score == 62
    assert risk.grade == "D"
