from __future__ import annotations

import os

import pytest

from clario360 import Clario360


def _integration_ready() -> bool:
    has_auth = bool(os.getenv("CLARIO360_API_KEY")) or bool(os.getenv("CLARIO360_ACCESS_TOKEN")) or (
        bool(os.getenv("CLARIO360_EMAIL")) and bool(os.getenv("CLARIO360_PASSWORD"))
    )
    return bool(os.getenv("CLARIO360_API_URL")) and has_auth


@pytest.mark.integration
@pytest.mark.skipif(not _integration_ready(), reason="integration environment is not configured")
def test_full_alert_workflow() -> None:
    client = Clario360()
    alerts = client.cyber.alerts.list(per_page=5)
    assert alerts.total >= 0
    if alerts.data:
        alert = client.cyber.alerts.get(alerts.data[0].id)
        assert alert.id == alerts.data[0].id
        assert alert.confidence_score >= 0
    client.close()
