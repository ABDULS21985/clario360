from __future__ import annotations

from typing import Any, Sequence

from clario360.exceptions import ConfigurationError
from clario360.models.risk import RiskTrendPoint


def plot_risk_trend(points: Sequence[RiskTrendPoint]) -> Any:
    try:
        import plotly.express as px
    except ImportError as exc:
        raise ConfigurationError(
            "plotly is required for plotting helpers. Install clario360[notebooks].",
            code="OPTIONAL_DEPENDENCY_MISSING",
        ) from exc

    return px.line(
        x=[item.recorded_at for item in points],
        y=[item.overall_score for item in points],
        title="Clario 360 Risk Trend",
        labels={"x": "Recorded at", "y": "Risk score"},
    )
