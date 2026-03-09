from __future__ import annotations

import base64
import json
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Tuple, Union

import pytest
from clario360.client import Clario360


def make_jwt(claims: Dict[str, Any]) -> str:
    header = _b64encode({"alg": "none", "typ": "JWT"})
    payload = _b64encode(claims)
    return f"{header}.{payload}.signature"


def _b64encode(value: Dict[str, Any]) -> str:
    raw = json.dumps(value, separators=(",", ":")).encode("utf-8")
    return base64.urlsafe_b64encode(raw).decode("utf-8").rstrip("=")


@dataclass
class RecordedRequest:
    method: str
    url: str
    params: Optional[Dict[str, Any]]
    json_body: Any
    headers: Dict[str, Any]


@dataclass
class MockResponse:
    status_code: int = 200
    payload: Any = None
    headers: Dict[str, str] = field(default_factory=lambda: {"Content-Type": "application/json"})
    text_override: Optional[str] = None
    reason: str = "OK"

    def json(self) -> Any:
        if self.payload is None:
            raise ValueError("response does not contain JSON")
        return self.payload

    @property
    def text(self) -> str:
        if self.text_override is not None:
            return self.text_override
        if self.payload is None:
            return ""
        return json.dumps(self.payload)

    @property
    def content(self) -> bytes:
        return self.text.encode("utf-8")


RouteResult = Union[MockResponse, Exception, Callable[[RecordedRequest], MockResponse]]


class StubRouter:
    def __init__(self) -> None:
        self.routes: Dict[Tuple[str, str], List[RouteResult]] = {}
        self.requests: List[RecordedRequest] = []

    def add(self, method: str, url: str, result: RouteResult) -> None:
        key = (method.upper(), url)
        self.routes.setdefault(key, []).append(result)

    def add_json(
        self,
        method: str,
        url: str,
        payload: Any,
        *,
        status_code: int = 200,
        headers: Optional[Dict[str, str]] = None,
    ) -> None:
        self.add(method, url, MockResponse(status_code=status_code, payload=payload, headers=headers or {"Content-Type": "application/json"}))

    def request(self, _: Any, method: str, url: str, **kwargs: Any) -> MockResponse:
        params = kwargs.get("params")
        json_body = kwargs.get("json")
        headers = kwargs.get("headers") or {}
        recorded = RecordedRequest(
            method=method.upper(),
            url=url,
            params=dict(params) if isinstance(params, dict) else params,
            json_body=json_body,
            headers=dict(headers),
        )
        self.requests.append(recorded)

        key = (method.upper(), url)
        if key not in self.routes or not self.routes[key]:
            raise AssertionError(f"unexpected request: {method.upper()} {url}")
        result = self.routes[key].pop(0)
        if callable(result):
            return result(recorded)
        if isinstance(result, Exception):
            raise result
        return result


@pytest.fixture()
def router(monkeypatch: pytest.MonkeyPatch) -> StubRouter:
    stub = StubRouter()
    def _request(session: Any, method: str, url: str, **kwargs: Any) -> MockResponse:
        return stub.request(session, method, url, **kwargs)

    monkeypatch.setattr("requests.sessions.Session.request", _request)
    monkeypatch.setattr("clario360.http_client.time.sleep", lambda _: None)

    async def _async_sleep(_: float) -> None:
        return None

    monkeypatch.setattr("clario360.http_client.asyncio.sleep", _async_sleep)
    return stub


@pytest.fixture()
def api_url() -> str:
    return "https://api.example.test"


@pytest.fixture()
def api_key_client(api_url: str) -> Clario360:
    return Clario360(api_url=api_url, api_key="clario360_ak_test")
