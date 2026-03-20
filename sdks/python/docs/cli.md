# CLI

Install the CLI extras:

```bash
pip install "clario360[cli]"
```

Configure the CLI:

```bash
clario360 config set api-url https://api.clario360.sa
clario360 config set api-key clario360_ak_example
```

Examples:

```bash
clario360 alerts list --severity critical
clario360 alerts ack ALERT_ID
clario360 risk score --output json
clario360 pipelines run PIPELINE_ID
```
