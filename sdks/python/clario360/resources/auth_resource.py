from __future__ import annotations

from typing import Optional

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.auth import AuthTokens, Session, User


class AuthResource:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self._http = http
        self._async_http = async_http

    def login(self) -> AuthTokens:
        return self._http.login()

    async def alogin(self) -> AuthTokens:
        return await self._async_http.login()

    def refresh(self) -> AuthTokens:
        return self._http.refresh()

    async def arefresh(self) -> AuthTokens:
        return await self._async_http.refresh()

    def logout(self) -> None:
        if self._http.auth_state.refresh_token:
            self._http.post("/api/v1/auth/logout", json={"refresh_token": self._http.auth_state.refresh_token})

    def whoami(self) -> User:
        payload = self._http.get("/api/v1/users/me")
        if isinstance(payload, dict) and "data" in payload and isinstance(payload["data"], dict):
            payload = payload["data"]
        user = User.from_dict(payload)
        self._http.set_user(user)
        return user

    def sessions(self) -> list[Session]:
        payload = self._http.get("/api/v1/users/me/sessions")
        if isinstance(payload, list):
            return [Session.from_dict(item) for item in payload if isinstance(item, dict)]
        return []

    async def awhoami(self) -> User:
        payload = await self._async_http.get("/api/v1/users/me")
        if isinstance(payload, dict) and "data" in payload and isinstance(payload["data"], dict):
            payload = payload["data"]
        return User.from_dict(payload)
