# Jupyter E2E Harness

This directory contains the real integration harness for the Secure AI Workspace:

- `run.sh` creates a kind cluster, deploys IAM, JupyterHub, Spark, and ClickHouse, then runs smoke checks.
- `smoke_test.py` exercises the deployed stack end to end, including notebook profile access control, template copy, persistence, network isolation, and a real Spark plus ClickHouse JDBC job from the spawned notebook pod.
- `run_oidc_conformance.sh` runs the OpenID Foundation conformance suite locally against the live IAM OIDC endpoints and drives the real Clario 360 login page during the authorization code flow.
- `values-e2e.yaml.tmpl` is the JupyterHub override template rendered by `run.sh`.
- `kind-config.yaml` is the kind cluster definition used by the harness.

Typical flow:

```bash
bash deploy/jupyter/e2e/run.sh
bash deploy/jupyter/e2e/run_oidc_conformance.sh
```

Environment knobs:

- `CLUSTER_NAME` override the kind cluster name.
- `KIND_BIN` override the kind binary path.
- `JUPYTER_PORT`, `IAM_PORT`, `FRONTEND_PORT` override local forwarded ports.
- `CONFORMANCE_VENV` overrides the Python virtualenv used by the conformance runner.
- `RESULTS_DIR` overrides the export directory for conformance suite HTML artifacts.
- `DELETE_CLUSTER=1` tears the cluster down after `run.sh`.
- `SKIP_IDLE_CULL=1` skips the idle-culler assertion in `smoke_test.py`.
