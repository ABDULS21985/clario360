class VCISOResource:
    def __init__(self, client):
        self.client = client

    def briefing(self):
        return self.client.get("/api/v1/cyber/vciso/briefing")

    def recommendations(self):
        return self.client.get("/api/v1/cyber/vciso/recommendations")
