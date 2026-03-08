class AssetResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/assets", params=params)

    def get(self, asset_id: str):
        return self.client.get(f"/api/v1/cyber/assets/{asset_id}")

    def create(self, **payload):
        return self.client.post("/api/v1/cyber/assets", json=payload)

    def bulk_import(self, items):
        return self.client.post("/api/v1/cyber/assets/bulk", json=items)

    def vulnerabilities(self, asset_id: str, **params):
        return self.client.list(f"/api/v1/cyber/assets/{asset_id}/vulnerabilities", params=params)
