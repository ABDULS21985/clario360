from __future__ import annotations

from typing import Callable, Iterator, TypeVar

from clario360.models.base import BaseModel
from clario360.models.common import PaginatedResponse

T = TypeVar("T", bound=BaseModel)


def iter_pages(fetch_page: Callable[[int], PaginatedResponse[T]]) -> Iterator[T]:
    page = 1
    while True:
        response = fetch_page(page)
        for item in response.data:
            yield item
        if page >= response.total_pages:
            break
        page += 1
