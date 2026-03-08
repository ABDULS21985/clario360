class SourceResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/data/sources", params=params)

    def discover(self, source_id: str):
        return self.client.post(f"/api/v1/data/sources/{source_id}/discover", json={})
