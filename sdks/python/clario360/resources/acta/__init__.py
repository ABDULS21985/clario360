from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.acta import ActionItem, Committee, Meeting
from clario360.resources.acta.action_items import ActionItemsResource
from clario360.resources.acta.committees import CommitteesResource
from clario360.resources.acta.meetings import MeetingsResource


class ActaNamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.committees = CommitteesResource(http, async_http, "/api/v1/acta/committees", Committee)
        self.meetings = MeetingsResource(http, async_http, "/api/v1/acta/meetings", Meeting)
        self.action_items = ActionItemsResource(http, async_http, "/api/v1/acta/action-items", ActionItem)


__all__ = ["ActaNamespace"]
