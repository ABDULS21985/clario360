#!/usr/bin/env python3

import json
import os
import subprocess
import sys
import time

import requests


IAM_BASE_URL = os.environ["IAM_BASE_URL"].rstrip("/")
JUPYTER_BASE_URL = os.environ["JUPYTER_BASE_URL"].rstrip("/")
TENANT_ID = os.environ["TEST_TENANT_ID"]
EMAIL = os.environ["TEST_EMAIL"]
PASSWORD = os.environ["TEST_PASSWORD"]
CLUSTER_NAME = os.environ["CLUSTER_NAME"]
SKIP_IDLE_CULL = os.environ.get("SKIP_IDLE_CULL") == "1"
VIEWER_EMAIL = os.environ.get("VIEWER_EMAIL", "viewer@example.com")
VIEWER_PASSWORD = os.environ.get("VIEWER_PASSWORD", "ViewerP@ssw0rd!2026")


def run(cmd: list[str], *, input_text: str | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        input=input_text,
        text=True,
        capture_output=True,
        check=check,
    )


def kubectl(*args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return run(["kubectl", "--context", f"kind-{CLUSTER_NAME}", *args], check=check)


def wait_http(method: str, url: str, *, expected=(200,), headers=None, json_body=None, timeout=180):
    deadline = time.time() + timeout
    last_error = None
    while time.time() < deadline:
        try:
            resp = requests.request(method, url, headers=headers, json=json_body, timeout=10, allow_redirects=True)
            if resp.status_code in expected:
                return resp
            last_error = RuntimeError(f"{method} {url} returned {resp.status_code}: {resp.text[:300]}")
        except Exception as exc:  # noqa: BLE001
            last_error = exc
        time.sleep(2)
    raise RuntimeError(f"timed out waiting for {url}: {last_error}")


def register_or_login(email: str, password: str, first_name: str, last_name: str) -> tuple[str, str]:
    payload = {
        "tenant_id": TENANT_ID,
        "email": email,
        "password": password,
        "first_name": first_name,
        "last_name": last_name,
    }
    resp = requests.post(f"{IAM_BASE_URL}/api/v1/auth/register", json=payload, timeout=15)
    if resp.status_code not in (201, 409):
        raise RuntimeError(f"register failed for {email}: {resp.status_code} {resp.text}")
    if resp.status_code == 201:
        data = resp.json()
        return data["access_token"], data["refresh_token"]

    resp = requests.post(
        f"{IAM_BASE_URL}/api/v1/auth/login",
        json={"tenant_id": TENANT_ID, "email": email, "password": password},
        timeout=15,
    )
    if resp.status_code != 200:
        raise RuntimeError(f"login failed for {email}: {resp.status_code} {resp.text}")
    data = resp.json()
    return data["access_token"], data["refresh_token"]


def auth_headers(access_token: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {access_token}", "Content-Type": "application/json"}


def assert_oidc_browser_flow(access_token: str) -> None:
    session = requests.Session()
    session.cookies.set("clario360_access", access_token, path="/")
    resp = session.get(f"{JUPYTER_BASE_URL}/hub/home", timeout=30, allow_redirects=True)
    if resp.status_code != 200:
        raise RuntimeError(f"jupyterhub home failed: {resp.status_code} {resp.text[:300]}")
    if not resp.url.startswith(JUPYTER_BASE_URL) or "/hub/" not in resp.url:
        raise RuntimeError(f"unexpected final JupyterHub URL: {resp.url}")


def assert_profiles_and_templates(access_token: str) -> None:
    profiles = requests.get(
        f"{IAM_BASE_URL}/api/v1/notebooks/profiles",
        headers=auth_headers(access_token),
        timeout=15,
    )
    if profiles.status_code != 200:
        raise RuntimeError(f"list notebook profiles failed: {profiles.status_code} {profiles.text}")
    slugs = {item["slug"] for item in profiles.json()}
    expected = {"soc-analyst", "data-scientist", "spark-connected", "admin"}
    if slugs != expected:
        raise RuntimeError(f"unexpected notebook profiles: {slugs}")

    templates = requests.get(
        f"{IAM_BASE_URL}/api/v1/notebooks/templates",
        headers=auth_headers(access_token),
        timeout=15,
    )
    if templates.status_code != 200:
        raise RuntimeError(f"list notebook templates failed: {templates.status_code} {templates.text}")
    data = templates.json()
    if len(data) != 10:
        raise RuntimeError(f"expected 10 notebook templates, got {len(data)}")
    if "01_threat_detection_quickstart" not in {item["id"] for item in data}:
        raise RuntimeError("missing threat detection quickstart template")


def assert_profile_access_control() -> None:
    viewer_token, _ = register_or_login(VIEWER_EMAIL, VIEWER_PASSWORD, "Viewer", "Analyst")
    profiles = requests.get(
        f"{IAM_BASE_URL}/api/v1/notebooks/profiles",
        headers=auth_headers(viewer_token),
        timeout=15,
    )
    if profiles.status_code != 200:
        raise RuntimeError(f"viewer profile list failed: {profiles.status_code} {profiles.text}")
    slugs = {item["slug"] for item in profiles.json()}
    if "admin" in slugs:
        raise RuntimeError(f"viewer should not see admin profile: {slugs}")

    denied = requests.post(
        f"{IAM_BASE_URL}/api/v1/notebooks/servers",
        headers=auth_headers(viewer_token),
        json={"profile": "admin"},
        timeout=20,
    )
    if denied.status_code != 403:
        raise RuntimeError(f"viewer admin profile start should be forbidden, got {denied.status_code}: {denied.text}")


def wait_server_running(access_token: str, timeout: int = 300) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = requests.get(
            f"{IAM_BASE_URL}/api/v1/notebooks/servers/default/status",
            headers=auth_headers(access_token),
            timeout=15,
        )
        if resp.status_code == 200 and resp.json().get("status") == "running":
            return
        time.sleep(5)
    raise RuntimeError("notebook server did not reach running state")


def get_notebook_pod() -> str:
    data = json.loads(kubectl("-n", "jupyterhub", "get", "pods", "-l", "component=singleuser-server", "-o", "json").stdout)
    items = data.get("items", [])
    if not items:
        raise RuntimeError("no notebook pod found")
    return items[0]["metadata"]["name"]


def exec_in_notebook(script: str) -> str:
    pod = get_notebook_pod()
    result = kubectl(
        "-n",
        "jupyterhub",
        "exec",
        pod,
        "--",
        "bash",
        "-lc",
        script,
    )
    return result.stdout


def assert_sdk_and_env() -> None:
    script = r"""
python - <<'PY'
from clario360 import Client
import json
import os
sdk = Client.from_env()
whoami = sdk.whoami()
print(json.dumps({
    "email": whoami["email"],
    "tenant_id": whoami["tenant_id"],
    "env_user_id": os.environ.get("CLARIO360_USER_ID"),
    "env_tenant_id": os.environ.get("CLARIO360_TENANT_ID"),
    "env_role": os.environ.get("CLARIO360_USER_ROLE"),
}))
PY
"""
    data = json.loads(exec_in_notebook(script).strip())
    if data["email"] != EMAIL:
        raise RuntimeError(f"unexpected notebook sdk identity: {data}")
    if data["tenant_id"] != TENANT_ID:
        raise RuntimeError(f"unexpected notebook tenant: {data}")
    if not data["env_user_id"] or data["env_tenant_id"] != TENANT_ID:
        raise RuntimeError(f"missing injected notebook env context: {data}")


def assert_copy_template(access_token: str) -> None:
    resp = requests.post(
        f"{IAM_BASE_URL}/api/v1/notebooks/servers/default/copy-template",
        headers=auth_headers(access_token),
        json={"template_id": "01_threat_detection_quickstart"},
        timeout=20,
    )
    if resp.status_code != 200:
        raise RuntimeError(f"copy notebook template failed: {resp.status_code} {resp.text}")
    copied = resp.json()
    if copied.get("template_id") != "01_threat_detection_quickstart":
        raise RuntimeError(f"unexpected copy-template response: {copied}")

    copied_file = exec_in_notebook("test -f /home/jovyan/01_threat_detection_quickstart.ipynb && echo present\n").strip()
    if copied_file != "present":
        raise RuntimeError("copied notebook template is not present in the notebook home directory")


def assert_network_policy() -> None:
    script = r"""
python - <<'PY'
import json
import socket
import urllib.request

def can_connect(host, port):
    sock = socket.socket()
    sock.settimeout(3)
    try:
        sock.connect((host, port))
        sock.close()
        return True
    except Exception:
        return False

def http_ok(url):
    try:
        with urllib.request.urlopen(url, timeout=5) as resp:
            return resp.status
    except Exception:
        return None

print(json.dumps({
    "kubernetes": can_connect("kubernetes.default.svc.cluster.local", 443),
    "redis": can_connect("redis.clario360.svc.cluster.local", 6379),
    "iam_http": http_ok("http://iam-service.clario360.svc.cluster.local:8081/.well-known/openid-configuration"),
    "clickhouse_http": http_ok("http://clickhouse.data.svc.cluster.local:8123/ping"),
}))
PY
"""
    result = json.loads(exec_in_notebook(script).strip())
    if result["kubernetes"]:
        raise RuntimeError("kubernetes API should be blocked by notebook network policy")
    if result["redis"]:
        raise RuntimeError("redis should be blocked by notebook network policy")
    if result["iam_http"] != 200:
        raise RuntimeError(f"iam-service should be reachable from notebooks: {result}")
    if result["clickhouse_http"] != 200:
        raise RuntimeError(f"clickhouse should be reachable from notebooks: {result}")


def assert_spark_and_clickhouse() -> None:
    script = r"""
python - <<'PY'
import json
import os
from pyspark.sql import SparkSession

jar = os.environ["CLICKHOUSE_JDBC_JAR"]
spark = (
    SparkSession.builder
    .appName("clario360-jupyter-e2e")
    .master(os.environ["SPARK_MASTER"])
    .config("spark.driver.host", os.environ["SPARK_DRIVER_HOST"])
    .config("spark.driver.bindAddress", os.environ["SPARK_DRIVER_BIND_ADDRESS"])
    .config("spark.driver.port", os.environ["SPARK_DRIVER_PORT"])
    .config("spark.blockManager.port", os.environ["SPARK_BLOCKMANAGER_PORT"])
    .config("spark.jars", jar)
    .config("spark.driver.extraClassPath", jar)
    .config("spark.executor.extraClassPath", jar)
    .getOrCreate()
)
try:
    word_counts = dict(
        spark.sparkContext.parallelize(["alpha beta", "beta gamma", "beta"])
        .flatMap(lambda line: line.split())
        .countByValue()
    )
    jdbc_url = "jdbc:clickhouse://clickhouse.data.svc.cluster.local:8123/default"
    df = (
        spark.read.format("jdbc")
        .option("url", jdbc_url)
        .option("driver", "com.clickhouse.jdbc.ClickHouseDriver")
        .option("dbtable", "(SELECT count(*) AS total FROM security_events) t")
        .option("user", open("/etc/clario360/credentials/clickhouse_user").read().strip())
        .option("password", open("/etc/clario360/credentials/clickhouse_password").read().strip())
        .load()
    )
    total = int(df.collect()[0]["total"])
    print(json.dumps({"word_counts": word_counts, "security_events": total}))
finally:
    spark.stop()
PY
"""
    result = json.loads(exec_in_notebook(script).strip())
    if result["word_counts"].get("beta") != 3:
        raise RuntimeError(f"unexpected Spark word count result: {result}")
    if result["security_events"] < 3:
        raise RuntimeError(f"unexpected ClickHouse event count: {result}")


def assert_persistence(access_token: str) -> None:
    exec_in_notebook("echo persistent-e2e > /home/jovyan/e2e-persist.txt\ncat /home/jovyan/e2e-persist.txt\n")
    stop_notebook(access_token)
    start_notebook(access_token)
    wait_server_running(access_token)
    contents = exec_in_notebook("cat /home/jovyan/e2e-persist.txt\n").strip()
    if contents != "persistent-e2e":
        raise RuntimeError("notebook home did not persist across restart")


def start_notebook(access_token: str, profile: str = "spark-connected") -> None:
    resp = requests.post(
        f"{IAM_BASE_URL}/api/v1/notebooks/servers",
        headers=auth_headers(access_token),
        json={"profile": profile},
        timeout=20,
    )
    if resp.status_code not in (201, 409):
        raise RuntimeError(f"start notebook failed: {resp.status_code} {resp.text}")


def stop_notebook(access_token: str) -> None:
    resp = requests.delete(
        f"{IAM_BASE_URL}/api/v1/notebooks/servers/default",
        headers=auth_headers(access_token),
        timeout=20,
    )
    if resp.status_code not in (200, 404):
        raise RuntimeError(f"stop notebook failed: {resp.status_code} {resp.text}")
    deadline = time.time() + 180
    while time.time() < deadline:
        pods = json.loads(kubectl("-n", "jupyterhub", "get", "pods", "-l", "component=singleuser-server", "-o", "json").stdout)
        if not pods.get("items"):
            return
        time.sleep(4)
    raise RuntimeError("notebook pod still exists after stop")


def assert_idle_cull(access_token: str) -> None:
    if SKIP_IDLE_CULL:
        return
    deadline = time.time() + 210
    while time.time() < deadline:
        resp = requests.get(
            f"{IAM_BASE_URL}/api/v1/notebooks/servers/default/status",
            headers=auth_headers(access_token),
            timeout=15,
        )
        if resp.status_code == 404:
            return
        time.sleep(10)
    raise RuntimeError("idle culler did not stop the notebook server")


def main() -> None:
    wait_http("GET", f"{IAM_BASE_URL}/.well-known/openid-configuration")
    wait_http("GET", f"{JUPYTER_BASE_URL}/hub/health")

    access_token, _ = register_or_login(EMAIL, PASSWORD, "Notebook", "Analyst")
    assert_profiles_and_templates(access_token)
    assert_profile_access_control()
    assert_oidc_browser_flow(access_token)

    stop_notebook(access_token)
    start_notebook(access_token)
    wait_server_running(access_token)

    assert_sdk_and_env()
    assert_copy_template(access_token)
    assert_network_policy()
    assert_spark_and_clickhouse()
    assert_persistence(access_token)
    assert_idle_cull(access_token)

    print(json.dumps({"status": "ok", "cluster": CLUSTER_NAME}))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:  # noqa: BLE001
        print(f"smoke test failed: {exc}", file=sys.stderr)
        raise
