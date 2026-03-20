from __future__ import annotations

from typing import Optional


def backoff_seconds(attempt: int, retry_after: Optional[int] = None) -> float:
    if retry_after is not None and retry_after > 0:
        return float(retry_after)
    return float(2 ** max(attempt - 1, 0))
