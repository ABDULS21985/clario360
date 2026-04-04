from __future__ import annotations

from typing import Optional


class LifecycleResource:
    def __init__(self, client):
        self.client = client

    def promote(self, model_id: str, version_id: str, approved_by: Optional[str] = None):
        payload = {}
        if approved_by:
            payload["approved_by"] = approved_by
        return self.client.post(f"/api/v1/ai/models/{model_id}/versions/{version_id}/promote", json=payload)

    def rollback(self, model_id: str, reason: str):
        return self.client.post(f"/api/v1/ai/models/{model_id}/rollback", json={"reason": reason})

    def shadow(self, model_id: str, action: str, version_id: Optional[str] = None, reason: Optional[str] = None):
        if action == "start":
            return self.client.post(f"/api/v1/ai/models/{model_id}/shadow/start", json={"version_id": version_id})
        if action == "stop":
            payload = {}
            if version_id:
                payload["version_id"] = version_id
            if reason:
                payload["reason"] = reason
            return self.client.post(f"/api/v1/ai/models/{model_id}/shadow/stop", json=payload)
        raise ValueError("action must be 'start' or 'stop'")
