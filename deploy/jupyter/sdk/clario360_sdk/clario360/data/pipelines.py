class PipelineResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/data/pipelines", params=params)

    def run(self, pipeline_id: str):
        return self.client.post(f"/api/v1/data/pipelines/{pipeline_id}/run", json={})
