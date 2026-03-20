from clario360.client import ListResult
from clario360 import Client
from clario360.utils.pagination import exhaust_pages


def test_exhaust_pages_single_page():
    client = Client("https://api.example.com", "token-123")
    result = ListResult(client, "/api/v1/test", {}, [{"id": "1"}], type("Meta", (), {"page": 1, "last_page": 1})())
    assert exhaust_pages(result) == [{"id": "1"}]
