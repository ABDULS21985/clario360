class QualityResource:
    def __init__(self, client):
        self.client = client

    def score(self):
        return self.client.get("/api/v1/data/quality/score")

    def rules(self, **params):
        return self.client.list("/api/v1/data/quality/rules", params=params)
