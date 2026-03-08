from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

import requests


@dataclass
class TokenBundle:
    access_token: str
    refresh_token: Optional[str] = None


class TokenManager:
    def __init__(self, api_url: str, access_token: str, refresh_token: Optional[str] = None):
        self.api_url = api_url.rstrip("/")
        self.access_token = access_token
        self.refresh_token = refresh_token

    def apply(self, headers: dict[str, str]) -> dict[str, str]:
        result = dict(headers)
        if self.access_token:
            result["Authorization"] = f"Bearer {self.access_token}"
        return result

    def refresh(self, session: requests.Session) -> TokenBundle:
        if not self.refresh_token:
            raise RuntimeError("refresh token is not configured")
        response = session.post(
            f"{self.api_url}/api/v1/auth/oauth/token",
            data={
                "grant_type": "refresh_token",
                "client_id": "jupyterhub",
                "refresh_token": self.refresh_token,
            },
            timeout=30,
        )
        response.raise_for_status()
        payload = response.json()
        self.access_token = payload["access_token"]
        self.refresh_token = payload.get("refresh_token", self.refresh_token)
        return TokenBundle(access_token=self.access_token, refresh_token=self.refresh_token)
