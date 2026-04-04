class AlertResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/alerts", params=params)

    def get(self, alert_id: str):
        return self.client.get(f"/api/v1/cyber/alerts/{alert_id}")

    def update_status(self, alert_id: str, status: str):
        return self.client.put(f"/api/v1/cyber/alerts/{alert_id}/status", json={"status": status})

    def comment(self, alert_id: str, content: str):
        return self.client.post(f"/api/v1/cyber/alerts/{alert_id}/comment", json={"content": content})

    def related(self, alert_id: str):
        return self.client.list(f"/api/v1/cyber/alerts/{alert_id}/related")
