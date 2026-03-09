from __future__ import annotations

from typing import TYPE_CHECKING, Union

if TYPE_CHECKING:
    from typing import Literal

StrPath = Union[str, bytes]

__all__ = ["StrPath"]
