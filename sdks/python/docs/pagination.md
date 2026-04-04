# Pagination

List endpoints return `PaginatedResponse[T]`.

```python
response = client.cyber.alerts.list(page=1, per_page=50)
print(response.total, response.total_pages)
for alert in response:
    print(alert.id)
```

For bulk iteration across every page:

```python
for alert in client.cyber.alerts.list_all(severity="critical"):
    print(alert.title)
```
