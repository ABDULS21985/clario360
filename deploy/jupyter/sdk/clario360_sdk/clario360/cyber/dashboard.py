class DashboardResource:
    def __init__(self, client):
        self.client = client

    def __call__(self):
        return self.client.get("/api/v1/cyber/dashboard")
