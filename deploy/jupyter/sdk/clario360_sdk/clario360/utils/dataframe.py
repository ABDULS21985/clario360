from __future__ import annotations

from typing import Iterable


def normalize_record(item):
    if hasattr(item, "items"):
        return dict(item)
    return item


def records_to_dataframe(items: Iterable):
    try:
        import pandas as pd
    except ModuleNotFoundError as exc:
        raise RuntimeError("pandas is required for DataFrame conversion") from exc
    return pd.DataFrame([normalize_record(item) for item in items])
