# Quickstart

```python
from clario360 import Clario360

client = Clario360()
alerts = client.cyber.alerts.list(per_page=5)
for alert in alerts:
    print(alert.title, alert.status)
client.close()
```

Set credentials through environment variables before running the snippet:

- `CLARIO360_API_URL`
- `CLARIO360_API_KEY` or `CLARIO360_ACCESS_TOKEN`
- `CLARIO360_EMAIL` and `CLARIO360_PASSWORD` for password login
