import os
from unittest.mock import patch

import pytest

from clario360 import Client
from clario360.client import ListResult


def test_client_from_env():
    with patch.dict(os.environ, {
        "CLARIO360_API_URL": "https://api.example.com",
        "CLARIO360_ACCESS_TOKEN": "token-123",
        "CLARIO360_REFRESH_TOKEN": "refresh-123",
    }, clear=True):
        client = Client.from_env()
        assert client.api_url == "https://api.example.com"
        assert client.tokens.access_token == "token-123"


def test_client_auth_header():
    client = Client("https://api.example.com", "token-123")
    headers = client.tokens.apply({})
    assert headers["Authorization"] == "Bearer token-123"


def test_whoami():
    client = Client("https://api.example.com", "token-123")
    with patch.object(client, "_request", return_value={"id": "u1", "email": "user@example.com"}) as mocked:
        whoami = client.whoami()
        assert whoami.email == "user@example.com"
        mocked.assert_called_once_with("GET", "/api/v1/users/me", params=None)


def test_auto_pagination():
    client = Client("https://api.example.com", "token-123")
    first = ListResult(client, "/api/v1/cyber/alerts", {"per_page": 2}, [{"id": "1"}, {"id": "2"}], type("Meta", (), {"page": 1, "last_page": 3})())
    second = ListResult(client, "/api/v1/cyber/alerts", {"per_page": 2, "page": 2}, [{"id": "3"}, {"id": "4"}], type("Meta", (), {"page": 2, "last_page": 3})())
    third = ListResult(client, "/api/v1/cyber/alerts", {"per_page": 2, "page": 3}, [{"id": "5"}], type("Meta", (), {"page": 3, "last_page": 3})())
    with patch.object(client, "list", side_effect=[second, third]):
        items = list(first.auto_paginate())
        assert len(items) == 5


def test_to_dataframe():
    pytest.importorskip("pandas")
    client = Client("https://api.example.com", "token-123")
    result = ListResult(client, "/api/v1/cyber/alerts", {}, [{"id": "1", "severity": "high"}], type("Meta", (), {"page": 1, "last_page": 1})())
    frame = result.to_dataframe()
    assert list(frame.columns) == ["id", "severity"]
    assert frame.iloc[0]["severity"] == "high"


def test_token_refresh():
    client = Client("https://api.example.com", "expired", "refresh-123")

    class FakeResponse:
      def __init__(self, status_code, payload):
        self.status_code = status_code
        self._payload = payload
        self.content = b"1"
      def json(self):
        return self._payload

    responses = [
        FakeResponse(401, {"error": "expired"}),
        FakeResponse(200, {"data": [{"id": "1"}], "pagination": {"page": 1, "last_page": 1, "total": 1}}),
    ]
    refresh_payload = {"access_token": "new-token", "refresh_token": "refresh-123"}

    def fake_request(method, url, **kwargs):
        return responses.pop(0)

    with patch.object(client.session, "request", side_effect=fake_request), patch.object(client.tokens, "refresh", return_value=refresh_payload):
        result = client.list("/api/v1/cyber/alerts")
        assert result.data[0]["id"] == "1"
