from __future__ import annotations

from typing import Any, Dict, Optional


class Clario360Error(Exception):
    """Base exception for SDK failures."""

    def __init__(
        self,
        message: str,
        *,
        code: str = "UNKNOWN",
        status_code: int = 0,
        details: Optional[Dict[str, Any]] = None,
    ) -> None:
        self.message = message
        self.code = code
        self.status_code = status_code
        self.details = details or {}
        super().__init__(message)


class ConfigurationError(Clario360Error):
    """Raised when SDK configuration is invalid or incomplete."""


class AuthenticationError(Clario360Error):
    """Raised for invalid or expired credentials."""


class PermissionError(Clario360Error):
    """Raised for permission-denied API responses."""


class NotFoundError(Clario360Error):
    """Raised when the requested resource does not exist."""


class ValidationError(Clario360Error):
    """Raised when request validation fails."""


class ConflictError(Clario360Error):
    """Raised when a resource conflicts with current system state."""


class RateLimitError(Clario360Error):
    """Raised when the API rate limits the caller."""

    def __init__(
        self,
        message: str = "Rate limit exceeded",
        *,
        retry_after: int = 30,
        details: Optional[Dict[str, Any]] = None,
    ) -> None:
        self.retry_after = retry_after
        super().__init__(
            message,
            code="RATE_LIMITED",
            status_code=429,
            details=details,
        )


class ServerError(Clario360Error):
    """Raised for server-side failures."""


class GovernanceError(Clario360Error):
    """Raised when governance approval rules block an operation."""
