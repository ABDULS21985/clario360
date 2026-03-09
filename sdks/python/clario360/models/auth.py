from __future__ import annotations

from typing import List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Role(BaseModel):
    id: str
    tenant_id: str
    name: str
    slug: str
    description: str = ""
    permissions: List[str] = Field(default_factory=list)
    is_system_role: bool | None = None
    created_at: str | None = None
    updated_at: str | None = None


class User(BaseModel):
    id: str
    tenant_id: str
    email: str
    first_name: str
    last_name: str
    full_name: str | None = None
    status: str
    mfa_enabled: bool = False
    last_login_at: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
    roles: List[Role] = Field(default_factory=list)


class AuthTokens(BaseModel):
    access_token: str
    refresh_token: str | None = None
    expires_at: str | None = None
    token_type: str = "Bearer"
    user: User | None = None


class Session(BaseModel):
    id: str
    user_agent: str = ""
    ip_address: str = ""
    created_at: str
    last_active_at: str
    is_current: bool = False


class APIKey(BaseModel):
    id: str
    name: str
    prefix: str | None = None
    created_at: str | None = None
    expires_at: str | None = None
    last_used_at: str | None = None
    revoked_at: str | None = None
