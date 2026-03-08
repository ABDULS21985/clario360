from unittest.mock import patch

from clario360 import Client


def test_alert_list_calls_expected_endpoint():
    client = Client("https://api.example.com", "token-123")
    with patch.object(client, "list", return_value="ok") as mocked:
        result = client.cyber.alerts.list(status="new")
        assert result == "ok"
        mocked.assert_called_once_with("/api/v1/cyber/alerts", params={"status": "new"})


def test_dashboard_calls_expected_endpoint():
    client = Client("https://api.example.com", "token-123")
    with patch.object(client, "get", return_value={"kpis": {}}) as mocked:
        result = client.cyber.dashboard()
        assert result["kpis"] == {}
        mocked.assert_called_once_with("/api/v1/cyber/dashboard")
