from __future__ import annotations

from click.testing import CliRunner

from clario360.models.common import PaginatedResponse
from clario360.models.cyber import Alert
from clario360.models.risk import RiskScore
from clario360_cli.cli import cli


class FakeAlertsResource:
    def list(self, **_: object) -> PaginatedResponse[Alert]:
        return PaginatedResponse.from_payload(
            {
                "data": [{"id": "alert-1", "title": "SQL Injection", "severity": "critical", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
                "meta": {"page": 1, "per_page": 25, "total": 1, "total_pages": 1},
            },
            Alert,
        )

    def acknowledge(self, _: str) -> Alert:
        return Alert(id="alert-1", title="SQL Injection", severity="critical", status="acknowledged", created_at="2026-03-09T00:00:00Z")


class FakeRiskResource:
    def score(self) -> RiskScore:
        return RiskScore(overall_score=62, grade="D", severity="high")


class FakeClient:
    def __init__(self, **_: object) -> None:
        self.cyber = type("CyberNamespace", (), {"alerts": FakeAlertsResource(), "risk": FakeRiskResource()})()

    def close(self) -> None:
        return None


def test_alerts_list_json(monkeypatch) -> None:
    monkeypatch.setattr("clario360_cli.cli.Clario360", FakeClient)
    runner = CliRunner()
    result = runner.invoke(cli, ["--output", "json", "alerts", "list"])
    assert result.exit_code == 0
    assert "SQL Injection" in result.output


def test_alerts_ack_table(monkeypatch) -> None:
    monkeypatch.setattr("clario360_cli.cli.Clario360", FakeClient)
    runner = CliRunner()
    result = runner.invoke(cli, ["alerts", "ack", "alert-1"])
    assert result.exit_code == 0
    assert "acknowledged" in result.output


def test_risk_score_json(monkeypatch) -> None:
    monkeypatch.setattr("clario360_cli.cli.Clario360", FakeClient)
    runner = CliRunner()
    result = runner.invoke(cli, ["--output", "json", "risk", "score"])
    assert result.exit_code == 0
    assert '"overall_score": 62.0' in result.output
