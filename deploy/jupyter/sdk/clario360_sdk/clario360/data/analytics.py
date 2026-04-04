class AnalyticsResource:
    def __init__(self, client):
        self.client = client

    def query(self, sql: str, **payload):
        body = {"query": sql}
        body.update(payload)
        return self.client.post("/api/v1/data/analytics/query", json=body)
