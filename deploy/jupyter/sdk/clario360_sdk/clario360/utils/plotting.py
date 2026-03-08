from __future__ import annotations

import plotly.express as px
import pandas as pd


def timeline(df: pd.DataFrame, x: str, y: str, color: str, title: str):
    fig = px.line(df, x=x, y=y, color=color, title=title)
    fig.update_layout(template="plotly_white")
    return fig


def severity_pie(df: pd.DataFrame, column: str = "severity", title: str = "Severity Distribution"):
    counts = df[column].value_counts()
    return px.pie(values=counts.values, names=counts.index, title=title)
