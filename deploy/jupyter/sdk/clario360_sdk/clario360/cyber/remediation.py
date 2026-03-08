class RemediationResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/remediation", params=params)

    def create(self, **payload):
        return self.client.post("/api/v1/cyber/remediation", json=payload)

    def dry_run(self, remediation_id: str):
        return self.client.post(f"/api/v1/cyber/remediation/{remediation_id}/dry-run", json={})

    def execute(self, remediation_id: str, **payload):
        return self.client.post(f"/api/v1/cyber/remediation/{remediation_id}/execute", json=payload)
