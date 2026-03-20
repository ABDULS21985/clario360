from __future__ import annotations

from clario360.models.acta import Meeting, Minutes
from clario360.models.common import PaginatedResponse
from clario360.resources._base import BaseResource


class MeetingsResource(BaseResource[Meeting]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Meeting]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, meeting_id: str) -> Meeting:
        return self._get(meeting_id)

    def create(self, payload: dict[str, object]) -> Meeting:
        return self._create(payload)

    def start(self, meeting_id: str) -> Meeting:
        return self._post_at(f"{self._base}/{meeting_id}/start", Meeting, {})

    def end(self, meeting_id: str) -> Meeting:
        return self._post_at(f"{self._base}/{meeting_id}/end", Meeting, {})

    def minutes(self, meeting_id: str) -> Minutes:
        return self._get_at(f"{self._base}/{meeting_id}/minutes", Minutes)
