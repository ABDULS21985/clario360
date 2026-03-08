class ThreatResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/threats", params=params)

    def check_indicators(self, indicators):
        return self.client.post("/api/v1/cyber/indicators/check", json={"indicators": indicators})
