class RuleResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/rules", params=params)

    def create(self, **payload):
        return self.client.post("/api/v1/cyber/rules", json=payload)

    def test(self, rule_id: str, **payload):
        return self.client.post(f"/api/v1/cyber/rules/{rule_id}/test", json=payload)
