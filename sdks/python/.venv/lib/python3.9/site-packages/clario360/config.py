from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Optional

from clario360._version import __version__
from clario360.exceptions import ConfigurationError


@dataclass(frozen=True)
class SDKConfig:
    api_url: str
    timeout: float = 30.0
    max_retries: int = 3
    verify_ssl: bool = True
    user_agent: str = f"clario360-python-sdk/{__version__}"

    @classmethod
    def from_values(
        cls,
        *,
        api_url: Optional[str],
        timeout: float,
        max_retries: int,
        verify_ssl: bool,
    ) -> "SDKConfig":
        resolved = api_url or os.getenv("CLARIO360_API_URL")
        if not resolved:
            raise ConfigurationError(
                "api_url is required. Set api_url explicitly or CLARIO360_API_URL in the environment.",
                code="CONFIGURATION_ERROR",
            )
        return cls(
            api_url=resolved.rstrip("/"),
            timeout=timeout,
            max_retries=max_retries,
            verify_ssl=verify_ssl,
        )
