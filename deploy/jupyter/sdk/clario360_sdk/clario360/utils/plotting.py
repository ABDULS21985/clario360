from __future__ import annotations

import matplotlib.pyplot as plt
import pandas as pd


def timeline(df: pd.DataFrame, x: str, y: str, color: str, title: str):
    fig, ax = plt.subplots()
    for series_name, group in df.groupby(color):
        ax.plot(group[x], group[y], label=str(series_name))
    ax.set_title(title)
    ax.set_xlabel(x)
    ax.set_ylabel(y)
    ax.legend()
    fig.autofmt_xdate()
    fig.tight_layout()
    return fig


def severity_pie(df: pd.DataFrame, column: str = "severity", title: str = "Severity Distribution"):
    counts = df[column].value_counts()
    fig, ax = plt.subplots()
    ax.pie(counts.values, labels=counts.index, autopct="%1.0f%%")
    ax.set_title(title)
    fig.tight_layout()
    return fig
