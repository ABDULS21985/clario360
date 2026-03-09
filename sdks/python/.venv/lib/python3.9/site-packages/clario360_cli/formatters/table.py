from __future__ import annotations

import csv
import io
from typing import Any, Dict, Iterable, List, cast

from clario360.models.common import PaginatedResponse
from clario360_cli.formatters.json_output import serialize_value


def _as_rows(value: Any) -> List[Dict[str, Any]]:
    serialized = serialize_value(value)
    if isinstance(value, PaginatedResponse):
        serialized = serialized.get("data", [])
    if isinstance(serialized, list):
        return [item for item in serialized if isinstance(item, dict)]
    if isinstance(serialized, dict):
        return [serialized]
    return [{"value": serialized}]


def render_table(value: Any) -> str:
    try:
        from tabulate import tabulate
    except ImportError as exc:
        raise RuntimeError("tabulate is required for table output. Install clario360[cli].") from exc

    rows = _as_rows(value)
    if not rows:
        return "No results."

    normalized = [{key: _stringify(cell) for key, cell in row.items()} for row in rows]
    return cast(str, tabulate(normalized, headers="keys", tablefmt="github"))


def render_csv(value: Any) -> str:
    rows = _as_rows(value)
    if not rows:
        return ""

    headers = _collect_headers(rows)
    buffer = io.StringIO()
    writer = csv.DictWriter(buffer, fieldnames=headers)
    writer.writeheader()
    for row in rows:
        writer.writerow({key: _stringify(row.get(key, "")) for key in headers})
    return buffer.getvalue().strip()


def _collect_headers(rows: Iterable[Dict[str, Any]]) -> List[str]:
    headers: List[str] = []
    for row in rows:
        for key in row.keys():
            if key not in headers:
                headers.append(key)
    return headers


def _stringify(value: Any) -> str:
    if isinstance(value, (list, dict)):
        return str(serialize_value(value))
    if value is None:
        return ""
    return str(value)
