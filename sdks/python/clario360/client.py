from __future__ import annotations

from typing import Optional

from clario360.auth import AuthState, decode_jwt_claims
from clario360.config import SDKConfig
from clario360.exceptions import ConfigurationError
from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.auth import User
from clario360.models.common import HealthStatus
from clario360.resources import AINamespace, ActaNamespace, AuthResource, CyberNamespace, DataNamespace, LexNamespace, VisusNamespace


class Clario360:
    """Official Python SDK client for the Clario 360 Enterprise AI Platform."""

    def __init__(
        self,
        *,
        api_url: str | None = None,
        api_key: str | None = None,
        email: str | None = None,
        password: str | None = None,
        access_token: str | None = None,
        refresh_token: str | None = None,
        timeout: float = 30.0,
        max_retries: int = 3,
        verify_ssl: bool = True,
    ) -> None:
        self.config = SDKConfig.from_values(
            api_url=api_url,
            timeout=timeout,
            max_retries=max_retries,
            verify_ssl=verify_ssl,
        )
        self.auth_state = AuthState.from_values(
            api_key=api_key,
            access_token=access_token,
            refresh_token=refresh_token,
            email=email,
            password=password,
        )
        self._http = HTTPClient(self.config, self.auth_state)
        self._async_http = AsyncHTTPClient(self.config, self.auth_state)

        self.auth = AuthResource(self._http, self._async_http)
        self.cyber = CyberNamespace(self._http, self._async_http)
        self.data = DataNamespace(self._http, self._async_http)
        self.acta = ActaNamespace(self._http, self._async_http)
        self.lex = LexNamespace(self._http, self._async_http)
        self.visus = VisusNamespace(self._http, self._async_http)
        self.ai = AINamespace(self._http, self._async_http)

    def close(self) -> None:
        self._http.close()

    async def aclose(self) -> None:
        await self._async_http.aclose()

    def whoami(self) -> User:
        return self.auth.whoami()

    async def awhoami(self) -> User:
        return await self.auth.awhoami()

    def health(self) -> HealthStatus:
        payload = self._http.get("/healthz")
        if isinstance(payload, dict):
            return HealthStatus.from_dict(payload)
        return HealthStatus(status="unknown")

    async def ahealth(self) -> HealthStatus:
        payload = await self._async_http.get("/healthz")
        if isinstance(payload, dict):
            return HealthStatus.from_dict(payload)
        return HealthStatus(status="unknown")

    @property
    def tenant_id(self) -> str:
        user = self._http.user
        if user is not None:
            return user.tenant_id
        if self.auth_state.access_token:
            claims = decode_jwt_claims(self.auth_state.access_token)
            tenant_id = claims.get("tid")
            if isinstance(tenant_id, str):
                return tenant_id
        user = self.whoami()
        if not user.tenant_id:
            raise ConfigurationError("tenant_id is unavailable for the current session", code="TENANT_ID_UNAVAILABLE")
        return user.tenant_id
