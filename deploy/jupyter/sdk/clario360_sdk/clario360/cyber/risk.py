class RiskResource:
    def __init__(self, client):
        self.client = client

    def score(self):
        return self.client.get("/api/v1/cyber/risk/score")

    def heatmap(self):
        return self.client.get("/api/v1/cyber/risk/heatmap")

    def recommendations(self):
        return self.client.get("/api/v1/cyber/risk/recommendations")
