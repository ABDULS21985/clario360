from __future__ import annotations

from typing import Iterable, Mapping, TYPE_CHECKING, Any, List

from clario360.exceptions import ConfigurationError

if TYPE_CHECKING:
    import pandas as pd


def models_to_dataframe(items: Iterable[Any]) -> "pd.DataFrame":
    try:
        import pandas as pd
    except ImportError as exc:
        raise ConfigurationError(
            "pandas is required for DataFrame export. Install clario360[notebooks].",
            code="OPTIONAL_DEPENDENCY_MISSING",
        ) from exc

    records: List[Mapping[str, Any]] = []
    for item in items:
        if hasattr(item, "model_dump"):
            records.append(item.model_dump(exclude_none=True))
        elif isinstance(item, dict):
            records.append(item)
        else:
            records.append({"value": item})
    return pd.DataFrame(records)
