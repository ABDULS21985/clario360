# Error Handling

The SDK raises typed exceptions instead of returning raw error payloads.

- `ValidationError` for `400`
- `AuthenticationError` for `401`
- `PermissionError` for `403`
- `GovernanceError` for governance-specific `403` responses
- `NotFoundError` for `404`
- `ConflictError` for `409`
- `RateLimitError` for `429`
- `ServerError` for `5xx`

Each exception includes the upstream error code, HTTP status, and the parsed `details` dictionary when the API provides one.
