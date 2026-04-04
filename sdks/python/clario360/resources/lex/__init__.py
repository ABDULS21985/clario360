from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.lex import ComplianceRule, Contract
from clario360.resources.lex.compliance import ComplianceResource
from clario360.resources.lex.contracts import ContractsResource


class LexNamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.contracts = ContractsResource(http, async_http, "/api/v1/lex/contracts", Contract)
        self.compliance = ComplianceResource(http, async_http, "/api/v1/lex/compliance/rules", ComplianceRule)


__all__ = ["LexNamespace"]
