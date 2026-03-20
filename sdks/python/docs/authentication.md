# Authentication

The SDK supports three authentication modes:

1. API key: `Clario360(api_url="...", api_key="...")`
2. Bearer token: `Clario360(api_url="...", access_token="...", refresh_token="...")`
3. Password login: `Clario360(api_url="...", email="...", password="...")`

Password login exchanges credentials at `/api/v1/auth/login`, stores the returned access token, and refreshes it through `/api/v1/auth/refresh` when a request returns `401`.
