class AssessmentResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/cyber/ctem/assessments", params=params)

    def findings(self, assessment_id: str, **params):
        return self.client.list(f"/api/v1/cyber/ctem/assessments/{assessment_id}/findings", params=params)


class CTEMResource:
    def __init__(self, client):
        self.client = client
        self.assessments = AssessmentResource(client)

    def findings(self, assessment_id: str, **params):
        return self.assessments.findings(assessment_id, **params)

    def exposure_score(self):
        return self.client.get("/api/v1/cyber/ctem/exposure-score")
