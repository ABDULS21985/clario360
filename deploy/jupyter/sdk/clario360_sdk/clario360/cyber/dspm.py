class DSPMResource:
    def __init__(self, client):
        self.client = client

    def data_assets(self, **params):
        return self.client.list("/api/v1/cyber/dspm/data-assets", params=params)

    def scan(self):
        return self.client.post("/api/v1/cyber/dspm/scan", json={})
