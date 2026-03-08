class PredictionResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/ai/predictions", params=params)

    def get(self, prediction_id: str):
        return self.client.get(f"/api/v1/ai/predictions/{prediction_id}")
