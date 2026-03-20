from __future__ import annotations

import base64
import json
import os
from dataclasses import dataclass
from typing import Any, Dict, Optional

from clario360.exceptions import ConfigurationError


@dataclass
class AuthState:
    api_key: Optional[str] = None
    access_token: Optional[str] = None
    refresh_token: Optional[str] = None
    email: Optional[str] = None
    password: Optional[str] = None
    tenant_id: Optional[str] = None

    @classmethod
    def from_values(
        cls,
        *,
        api_key: Optional[str],
        access_token: Optional[str],
        refresh_token: Optional[str],
        email: Optional[str],
        password: Optional[str],
    ) -> "AuthState":
        state = cls(
            api_key=api_key or os.getenv("CLARIO360_API_KEY"),
            access_token=access_token or os.getenv("CLARIO360_ACCESS_TOKEN"),
            refresh_token=refresh_token or os.getenv("CLARIO360_REFRESH_TOKEN"),
            email=email or os.getenv("CLARIO360_EMAIL"),
            password=password or os.getenv("CLARIO360_PASSWORD"),
        )
        if state.api_key:
            return state
        if state.access_token:
            return state
        if state.email and state.password:
            return state
        raise ConfigurationError(
            "No authentication method configured. Supply api_key, access_token, or email/password.",
            code="CONFIGURATION_ERROR",
        )

    def has_api_key(self) -> bool:
        return bool(self.api_key)

    def has_access_token(self) -> bool:
        return bool(self.access_token)

    def can_login(self) -> bool:
        return bool(self.email and self.password)

    def can_refresh(self) -> bool:
        return bool(self.refresh_token and not self.api_key)

    def auth_headers(self) -> Dict[str, str]:
        if self.api_key:
            return {"X-API-Key": self.api_key}
        if self.access_token:
            return {"Authorization": f"Bearer {self.access_token}"}
        return {}

    def set_tokens(self, access_token: str, refresh_token: Optional[str]) -> None:
        self.access_token = access_token
        if refresh_token:
            self.refresh_token = refresh_token


def decode_jwt_claims(token: str) -> Dict[str, Any]:
    """Decode JWT claims without signature verification for local claim access."""

    parts = token.split(".")
    if len(parts) < 2:
        raise ConfigurationError("Invalid JWT format", code="INVALID_TOKEN")
    payload = parts[1]
    padding = "=" * ((4 - len(payload) % 4) % 4)
    decoded = base64.urlsafe_b64decode(payload + padding)
    claims = json.loads(decoded.decode("utf-8"))
    if not isinstance(claims, dict):
        raise ConfigurationError("JWT claims payload is invalid", code="INVALID_TOKEN")
    return claims
