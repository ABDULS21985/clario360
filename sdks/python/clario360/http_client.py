from __future__ import annotations

import asyncio
import json
import time
from typing import Any, Dict, Mapping, MutableMapping, Optional

import requests
from requests import Response

from clario360.auth import AuthState
from clario360.config import SDKConfig
from clario360.exceptions import (
    AuthenticationError,
    Clario360Error,
    ConflictError,
    GovernanceError,
    NotFoundError,
    PermissionError,
    RateLimitError,
    ServerError,
    ValidationError,
)
from clario360.models.auth import AuthTokens, User
from clario360.utils.retry import backoff_seconds


JSONValue = Any


class HTTPClient:
    """Typed sync HTTP transport for the Clario 360 SDK."""

    def __init__(self, config: SDKConfig, auth: AuthState) -> None:
        self._config = config
        self._auth = auth
        self._session = requests.Session()
        self._user: Optional[User] = None
        if self._auth.can_login() and not self._auth.has_access_token() and not self._auth.has_api_key():
            self.login()

    @property
    def user(self) -> Optional[User]:
        return self._user

    @property
    def auth_state(self) -> AuthState:
        return self._auth

    def set_user(self, user: User) -> None:
        self._user = user

    def close(self) -> None:
        self._session.close()

    def login(self) -> AuthTokens:
        if not self._auth.can_login():
            raise AuthenticationError("email and password are required for login", code="LOGIN_REQUIRED")
        payload = {
            "email": self._auth.email,
            "password": self._auth.password,
        }
        if self._auth.tenant_id:
            payload["tenant_id"] = self._auth.tenant_id
        data = self._request("POST", "/api/v1/auth/login", json=payload, include_auth=False, allow_refresh=False)
        tokens = AuthTokens.from_dict(self._coerce_mapping(data))
        self._auth.set_tokens(tokens.access_token, tokens.refresh_token)
        self._user = tokens.user
        if tokens.user is not None:
            self._auth.tenant_id = tokens.user.tenant_id
        return tokens

    def refresh(self) -> AuthTokens:
        if not self._auth.can_refresh():
            raise AuthenticationError("refresh token is not available", code="REFRESH_UNAVAILABLE")
        data = self._request(
            "POST",
            "/api/v1/auth/refresh",
            json={"refresh_token": self._auth.refresh_token},
            include_auth=False,
            allow_refresh=False,
        )
        tokens = AuthTokens.from_dict(self._coerce_mapping(data))
        self._auth.set_tokens(tokens.access_token, tokens.refresh_token)
        self._user = tokens.user
        if tokens.user is not None:
            self._auth.tenant_id = tokens.user.tenant_id
        return tokens

    def get(self, path: str, *, params: Optional[Mapping[str, Any]] = None) -> JSONValue:
        return self._request("GET", path, params=params)

    def post(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
        params: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return self._request("POST", path, params=params, json=json)

    def put(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return self._request("PUT", path, json=json)

    def patch(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return self._request("PATCH", path, json=json)

    def delete(self, path: str) -> JSONValue:
        return self._request("DELETE", path)

    def _request(
        self,
        method: str,
        path: str,
        *,
        params: Optional[Mapping[str, Any]] = None,
        json: Optional[Mapping[str, Any]] = None,
        include_auth: bool = True,
        allow_refresh: bool = True,
    ) -> JSONValue:
        url = path if path.startswith("http") else f"{self._config.api_url}{path}"
        headers: Dict[str, str] = {
            "Accept": "application/json",
            "User-Agent": self._config.user_agent,
        }
        if json is not None:
            headers["Content-Type"] = "application/json"
        if include_auth:
            headers.update(self._auth.auth_headers())

        refreshed = False
        attempt = 0
        while True:
            try:
                response = self._session.request(
                    method=method,
                    url=url,
                    params=self._clean_params(params),
                    json=json,
                    headers=headers,
                    timeout=self._config.timeout,
                    verify=self._config.verify_ssl,
                )
            except requests.Timeout as exc:
                if attempt >= self._config.max_retries:
                    raise Clario360Error("request timed out", code="TIMEOUT") from exc
                attempt += 1
                time.sleep(backoff_seconds(attempt))
                continue
            except requests.RequestException as exc:
                if attempt >= self._config.max_retries:
                    raise Clario360Error("connection error", code="NETWORK_ERROR") from exc
                attempt += 1
                time.sleep(backoff_seconds(attempt))
                continue

            if response.status_code == 401 and allow_refresh and not refreshed and self._auth.can_refresh():
                self.refresh()
                headers.update(self._auth.auth_headers())
                refreshed = True
                continue

            if response.status_code == 429 and attempt < self._config.max_retries:
                attempt += 1
                retry_after = self._retry_after(response)
                time.sleep(backoff_seconds(attempt, retry_after))
                continue

            if response.status_code >= 500 and attempt < self._config.max_retries:
                attempt += 1
                time.sleep(backoff_seconds(attempt))
                continue

            if response.status_code >= 400:
                self._raise_for_response(response)

            return self._parse_response(response)

    def _parse_response(self, response: Response) -> JSONValue:
        if not response.content:
            return {}
        content_type = response.headers.get("Content-Type", "")
        if "application/json" in content_type:
            return response.json()
        return response.text

    def _raise_for_response(self, response: Response) -> None:
        payload = self._parse_error_payload(response)
        message = payload.get("message", f"HTTP {response.status_code}")
        code = payload.get("code", "UNKNOWN")
        details = payload.get("details")

        if response.status_code == 400:
            raise ValidationError(message, code=code, status_code=400, details=details)
        if response.status_code == 401:
            raise AuthenticationError(message, code=code, status_code=401, details=details)
        if response.status_code == 403:
            if "GOVERNANCE" in code.upper():
                raise GovernanceError(message, code=code, status_code=403, details=details)
            raise PermissionError(message, code=code, status_code=403, details=details)
        if response.status_code == 404:
            raise NotFoundError(message, code=code, status_code=404, details=details)
        if response.status_code == 409:
            raise ConflictError(message, code=code, status_code=409, details=details)
        if response.status_code == 429:
            raise RateLimitError(message, retry_after=self._retry_after(response) or 30, details=details)
        raise ServerError(message, code=code, status_code=response.status_code, details=details)

    def _parse_error_payload(self, response: Response) -> Dict[str, Any]:
        try:
            payload = response.json()
        except ValueError:
            return {"message": response.text or f"HTTP {response.status_code}", "code": "UNKNOWN"}
        if isinstance(payload, dict):
            error = payload.get("error")
            if isinstance(error, dict):
                return {
                    "message": str(error.get("message", response.reason)),
                    "code": str(error.get("code", "UNKNOWN")),
                    "details": error.get("details") if isinstance(error.get("details"), dict) else {},
                }
            return {
                "message": str(payload.get("message", response.reason)),
                "code": str(payload.get("code", "UNKNOWN")),
                "details": payload.get("details") if isinstance(payload.get("details"), dict) else {},
            }
        return {"message": response.text or f"HTTP {response.status_code}", "code": "UNKNOWN"}

    def _retry_after(self, response: Response) -> Optional[int]:
        raw = response.headers.get("Retry-After")
        if raw is None:
            return None
        try:
            return int(raw)
        except ValueError:
            return None

    def _clean_params(self, params: Optional[Mapping[str, Any]]) -> Optional[Dict[str, Any]]:
        if params is None:
            return None
        cleaned: Dict[str, Any] = {}
        for key, value in params.items():
            if value is None:
                continue
            cleaned[key] = value
        return cleaned

    def _coerce_mapping(self, payload: JSONValue) -> Dict[str, Any]:
        if not isinstance(payload, dict):
            raise Clario360Error("expected object response", code="INVALID_RESPONSE")
        return payload


class AsyncHTTPClient:
    """Optional async transport that mirrors the sync client behaviour."""

    def __init__(self, config: SDKConfig, auth: AuthState) -> None:
        self._config = config
        self._auth = auth
        self._user: Optional[User] = None
        self._httpx: Any | None = None
        self._client: Any | None = None
        try:
            import httpx
        except ImportError:
            self._httpx = None
            self._client = None
            return
        self._httpx = httpx
        self._client = httpx.AsyncClient(verify=config.verify_ssl, timeout=config.timeout)
        if self._auth.can_login() and not self._auth.has_access_token() and not self._auth.has_api_key():
            # Login is deferred to the first async call so core installs do not require httpx.
            self._user = None

    @property
    def user(self) -> Optional[User]:
        return self._user

    def _ensure_client(self) -> Any:
        if self._httpx is None or self._client is None:
            raise Clario360Error(
                "Async support requires httpx. Install clario360[async].",
                code="OPTIONAL_DEPENDENCY_MISSING",
            )
        return self._client

    async def aclose(self) -> None:
        client = self._client
        if client is not None:
            await client.aclose()

    async def login(self) -> AuthTokens:
        if not self._auth.can_login():
            raise AuthenticationError("email and password are required for login", code="LOGIN_REQUIRED")
        payload: Dict[str, Any] = {
            "email": self._auth.email,
            "password": self._auth.password,
        }
        if self._auth.tenant_id:
            payload["tenant_id"] = self._auth.tenant_id
        data = await self._request(
            "POST",
            "/api/v1/auth/login",
            json=payload,
            include_auth=False,
            allow_refresh=False,
        )
        tokens = AuthTokens.from_dict(self._coerce_mapping(data))
        self._auth.set_tokens(tokens.access_token, tokens.refresh_token)
        self._user = tokens.user
        if tokens.user is not None:
            self._auth.tenant_id = tokens.user.tenant_id
        return tokens

    async def refresh(self) -> AuthTokens:
        await self._refresh()
        if not self._auth.access_token:
            raise AuthenticationError("refresh did not return an access token", code="INVALID_RESPONSE")
        return AuthTokens(
            access_token=self._auth.access_token,
            refresh_token=self._auth.refresh_token,
            user=self._user,
        )

    async def get(self, path: str, *, params: Optional[Mapping[str, Any]] = None) -> JSONValue:
        return await self._request("GET", path, params=params)

    async def post(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
        params: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return await self._request("POST", path, params=params, json=json)

    async def put(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return await self._request("PUT", path, json=json)

    async def patch(
        self,
        path: str,
        *,
        json: Optional[Mapping[str, Any]] = None,
    ) -> JSONValue:
        return await self._request("PATCH", path, json=json)

    async def delete(self, path: str) -> JSONValue:
        return await self._request("DELETE", path)

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: Optional[Mapping[str, Any]] = None,
        json: Optional[Mapping[str, Any]] = None,
        include_auth: bool = True,
        allow_refresh: bool = True,
    ) -> JSONValue:
        client = self._ensure_client()
        if self._auth.can_login() and not self._auth.has_access_token() and not self._auth.has_api_key():
            await self.login()
        url = path if path.startswith("http") else f"{self._config.api_url}{path}"
        headers: Dict[str, str] = {
            "Accept": "application/json",
            "User-Agent": self._config.user_agent,
        }
        if json is not None:
            headers["Content-Type"] = "application/json"
        if include_auth:
            headers.update(self._auth.auth_headers())

        refreshed = False
        attempt = 0
        while True:
            try:
                response = await client.request(
                    method,
                    url,
                    params=self._clean_params(params),
                    json=json,
                    headers=headers,
                )
            except self._httpx.TimeoutException as exc:
                if attempt >= self._config.max_retries:
                    raise Clario360Error("request timed out", code="TIMEOUT") from exc
                attempt += 1
                await asyncio.sleep(backoff_seconds(attempt))
                continue
            except self._httpx.HTTPError as exc:
                if attempt >= self._config.max_retries:
                    raise Clario360Error("connection error", code="NETWORK_ERROR") from exc
                attempt += 1
                await asyncio.sleep(backoff_seconds(attempt))
                continue

            if response.status_code == 401 and allow_refresh and not refreshed and self._auth.can_refresh():
                await self._refresh()
                headers.update(self._auth.auth_headers())
                refreshed = True
                continue

            if response.status_code == 429 and attempt < self._config.max_retries:
                attempt += 1
                await asyncio.sleep(backoff_seconds(attempt, self._retry_after(response)))
                continue

            if response.status_code >= 500 and attempt < self._config.max_retries:
                attempt += 1
                await asyncio.sleep(backoff_seconds(attempt))
                continue

            if response.status_code >= 400:
                self._raise_for_response(response.status_code, response.text, response.headers)

            if not response.content:
                return {}
            if "application/json" in response.headers.get("Content-Type", ""):
                return response.json()
            return response.text

    async def _refresh(self) -> None:
        if not self._auth.can_refresh():
            raise AuthenticationError("refresh token is not available", code="REFRESH_UNAVAILABLE")
        payload = await self._request(
            "POST",
            "/api/v1/auth/refresh",
            json={"refresh_token": self._auth.refresh_token},
            include_auth=False,
            allow_refresh=False,
        )
        tokens = AuthTokens.from_dict(self._coerce_mapping(payload))
        self._auth.set_tokens(tokens.access_token, tokens.refresh_token)
        self._user = tokens.user
        if tokens.user is not None:
            self._auth.tenant_id = tokens.user.tenant_id

    def _retry_after(self, response: Any) -> Optional[int]:
        raw = response.headers.get("Retry-After")
        if raw is None:
            return None
        try:
            return int(raw)
        except ValueError:
            return None

    def _raise_for_response(self, status_code: int, text: str, headers: MutableMapping[str, str]) -> None:
        payload: Dict[str, Any]
        try:
            parsed = json.loads(text)
        except ValueError:
            parsed = {"message": text or f"HTTP {status_code}", "code": "UNKNOWN"}
        if isinstance(parsed, dict) and isinstance(parsed.get("error"), dict):
            error = parsed["error"]
            payload = {
                "message": str(error.get("message", "")),
                "code": str(error.get("code", "UNKNOWN")),
                "details": error.get("details") if isinstance(error.get("details"), dict) else {},
            }
        elif isinstance(parsed, dict):
            payload = {
                "message": str(parsed.get("message", "")),
                "code": str(parsed.get("code", "UNKNOWN")),
                "details": parsed.get("details") if isinstance(parsed.get("details"), dict) else {},
            }
        else:
            payload = {"message": text or f"HTTP {status_code}", "code": "UNKNOWN", "details": {}}

        message = payload["message"] or f"HTTP {status_code}"
        code = payload["code"]
        details = payload["details"]

        if status_code == 400:
            raise ValidationError(message, code=code, status_code=status_code, details=details)
        if status_code == 401:
            raise AuthenticationError(message, code=code, status_code=status_code, details=details)
        if status_code == 403:
            if "GOVERNANCE" in code.upper():
                raise GovernanceError(message, code=code, status_code=status_code, details=details)
            raise PermissionError(message, code=code, status_code=status_code, details=details)
        if status_code == 404:
            raise NotFoundError(message, code=code, status_code=status_code, details=details)
        if status_code == 409:
            raise ConflictError(message, code=code, status_code=status_code, details=details)
        if status_code == 429:
            raise RateLimitError(
                message,
                retry_after=self._retry_after(type("R", (), {"headers": headers})()) or 30,
                details=details,
            )
        raise ServerError(message, code=code, status_code=status_code, details=details)

    def _clean_params(self, params: Optional[Mapping[str, Any]]) -> Optional[Dict[str, Any]]:
        if params is None:
            return None
        cleaned: Dict[str, Any] = {}
        for key, value in params.items():
            if value is None:
                continue
            cleaned[key] = value
        return cleaned

    def _coerce_mapping(self, payload: JSONValue) -> Dict[str, Any]:
        if not isinstance(payload, dict):
            raise Clario360Error("expected object response", code="INVALID_RESPONSE")
        return payload
