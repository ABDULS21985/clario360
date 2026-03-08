#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
E2E_DIR="${ROOT_DIR}/deploy/jupyter/e2e"
CLUSTER_NAME="${CLUSTER_NAME:-clario360-jupyter-e2e}"
KIND_BIN="${KIND_BIN:-/tmp/kind}"
JUPYTER_PORT="${JUPYTER_PORT:-38080}"
IAM_PORT="${IAM_PORT:-38081}"
ENV_FILE="${ENV_FILE:-/tmp/clario360-jupyter-e2e.env}"

POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-$(openssl rand -hex 16)}"
JUPYTERHUB_DB_PASSWORD="${JUPYTERHUB_DB_PASSWORD:-${POSTGRES_PASSWORD}}"
JUPYTERHUB_CLIENT_SECRET="${JUPYTERHUB_CLIENT_SECRET:-$(openssl rand -hex 24)}"
JUPYTERHUB_ADMIN_TOKEN="${JUPYTERHUB_ADMIN_TOKEN:-$(openssl rand -hex 24)}"
CLICKHOUSE_USER="${CLICKHOUSE_USER:-clario360}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-$(openssl rand -hex 18)}"
TEST_TENANT_ID="${TEST_TENANT_ID:-11111111-1111-1111-1111-111111111111}"
TEST_EMAIL="${TEST_EMAIL:-analyst@example.com}"
TEST_PASSWORD="${TEST_PASSWORD:-NotebookP@ssw0rd!2026}"
CONFORMANCE_CLIENT_SECRET="${CONFORMANCE_CLIENT_SECRET:-$(openssl rand -hex 24)}"

TEMP_DIR="$(mktemp -d)"
IAM_PORT_FORWARD_PID=""
JUPYTER_PORT_FORWARD_PID=""

cleanup() {
  if [[ -n "${IAM_PORT_FORWARD_PID}" ]]; then
    kill "${IAM_PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${JUPYTER_PORT_FORWARD_PID}" ]]; then
    kill "${JUPYTER_PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TEMP_DIR}"
  if [[ "${DELETE_CLUSTER:-0}" == "1" ]]; then
    "${KIND_BIN}" delete cluster --name "${CLUSTER_NAME}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

log() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

wait_for_rollout() {
  kubectl --context "kind-${CLUSTER_NAME}" -n "$1" rollout status "deployment/$2" --timeout=300s
}

wait_for_http() {
  local url="$1"
  local tries="${2:-90}"
  for _ in $(seq 1 "${tries}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for ${url}" >&2
  return 1
}

render_values() {
  local target="${TEMP_DIR}/values-e2e.yaml"
  sed \
    -e "s|__POSTGRES_PASSWORD__|${JUPYTERHUB_DB_PASSWORD}|g" \
    -e "s|__JUPYTERHUB_CLIENT_SECRET__|${JUPYTERHUB_CLIENT_SECRET}|g" \
    -e "s|__JUPYTERHUB_ADMIN_TOKEN__|${JUPYTERHUB_ADMIN_TOKEN}|g" \
    -e "s|__JUPYTER_PORT__|${JUPYTER_PORT}|g" \
    -e "s|__IAM_PORT__|${IAM_PORT}|g" \
    "${E2E_DIR}/values-e2e.yaml.tmpl" > "${target}"
  printf '%s' "${target}"
}

write_env_file() {
  cat > "${ENV_FILE}" <<EOF
export CLUSTER_NAME='${CLUSTER_NAME}'
export IAM_BASE_URL='http://localhost:${IAM_PORT}'
export JUPYTER_BASE_URL='http://localhost:${JUPYTER_PORT}'
export FRONTEND_BASE_URL='http://localhost:33000'
export TEST_TENANT_ID='${TEST_TENANT_ID}'
export TEST_EMAIL='${TEST_EMAIL}'
export TEST_PASSWORD='${TEST_PASSWORD}'
export CONFORMANCE_CLIENT_SECRET='${CONFORMANCE_CLIENT_SECRET}'
EOF
}

require_cmd docker
require_cmd kubectl
require_cmd helm
require_cmd curl
require_cmd python3
[[ -x "${KIND_BIN}" ]] || {
  echo "kind binary not found or not executable at ${KIND_BIN}" >&2
  exit 1
}

log "Creating or reusing kind cluster ${CLUSTER_NAME}"
if ! "${KIND_BIN}" get clusters | grep -qx "${CLUSTER_NAME}"; then
  "${KIND_BIN}" create cluster --name "${CLUSTER_NAME}" --config "${E2E_DIR}/kind-config.yaml"
fi

log "Building local images"
docker build -f "${ROOT_DIR}/deploy/docker/Dockerfile.backend" --build-arg BUILD_SERVICE=iam-service -t clario360/iam-service:e2e "${ROOT_DIR}"
docker build -f "${ROOT_DIR}/deploy/docker/Dockerfile.migrator" -t clario360/migrator:e2e "${ROOT_DIR}"
docker build -f "${ROOT_DIR}/deploy/jupyter/docker/Dockerfile.notebook" -t clario360/notebook:e2e "${ROOT_DIR}"
docker build -f "${ROOT_DIR}/deploy/jupyter/docker/Dockerfile.notebook-spark" --build-arg BASE_NOTEBOOK_IMAGE=clario360/notebook:e2e -t clario360/notebook-spark:e2e "${ROOT_DIR}"

log "Loading images into kind"
"${KIND_BIN}" load docker-image --name "${CLUSTER_NAME}" \
  clario360/iam-service:e2e \
  clario360/migrator:e2e \
  clario360/notebook:e2e \
  clario360/notebook-spark:e2e

log "Applying namespaces and cluster resources"
kubectl --context "kind-${CLUSTER_NAME}" apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: clario360
  labels:
    name: clario360
---
apiVersion: v1
kind: Namespace
metadata:
  name: data
  labels:
    name: data
---
apiVersion: v1
kind: Namespace
metadata:
  name: spark
  labels:
    name: spark
---
apiVersion: v1
kind: Namespace
metadata:
  name: jupyterhub
  labels:
    name: jupyterhub
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: jupyterhub-user-home-pv
spec:
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: ""
  hostPath:
    path: /var/lib/clario360-jupyter-home
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jupyterhub-user-home
  namespace: jupyterhub
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: ""
  volumeName: jupyterhub-user-home-pv
EOF

log "Deploying Postgres, Redis, ClickHouse, Spark, and IAM"
kubectl --context "kind-${CLUSTER_NAME}" apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: platform-secrets
  namespace: clario360
stringData:
  postgres-password: "${POSTGRES_PASSWORD}"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-init
  namespace: clario360
data:
  01-init.sql: |
    CREATE DATABASE platform_core;
    CREATE DATABASE jupyterhub;
    GRANT ALL PRIVILEGES ON DATABASE platform_core TO clario;
    GRANT ALL PRIVILEGES ON DATABASE jupyterhub TO clario;
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgresql
  namespace: clario360
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
        - name: postgresql
          image: postgres:16-alpine
          env:
            - name: POSTGRES_USER
              value: clario
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: platform-secrets
                  key: postgres-password
            - name: POSTGRES_DB
              value: postgres
          ports:
            - containerPort: 5432
          readinessProbe:
            exec:
              command: ["sh", "-c", "pg_isready -U clario -d postgres"]
            initialDelaySeconds: 10
            periodSeconds: 5
          volumeMounts:
            - name: init
              mountPath: /docker-entrypoint-initdb.d
      volumes:
        - name: init
          configMap:
            name: postgres-init
---
apiVersion: v1
kind: Service
metadata:
  name: postgresql
  namespace: clario360
spec:
  selector:
    app: postgresql
  ports:
    - name: postgres
      port: 5432
      targetPort: 5432
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: clario360
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:7.2-alpine
          ports:
            - containerPort: 6379
          readinessProbe:
            exec:
              command: ["redis-cli", "ping"]
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: clario360
spec:
  selector:
    app: redis
  ports:
    - name: redis
      port: 6379
      targetPort: 6379
---
apiVersion: v1
kind: Secret
metadata:
  name: clickhouse-credentials
  namespace: data
stringData:
  username: "${CLICKHOUSE_USER}"
  password: "${CLICKHOUSE_PASSWORD}"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: clickhouse-init
  namespace: data
data:
  01-security-events.sql: |
    CREATE TABLE IF NOT EXISTS security_events (
      timestamp DateTime,
      action String,
      source_ip String,
      user_id String
    ) ENGINE = MergeTree()
    ORDER BY (timestamp, source_ip);
    INSERT INTO security_events (timestamp, action, source_ip, user_id) VALUES
      (now() - INTERVAL 5 minute, 'login_failure', '10.0.0.1', 'user-a'),
      (now() - INTERVAL 4 minute, 'login_failure', '10.0.0.1', 'user-a'),
      (now() - INTERVAL 3 minute, 'login_success', '10.0.0.1', 'user-a');
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clickhouse
  namespace: data
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clickhouse
  template:
    metadata:
      labels:
        app: clickhouse
    spec:
      containers:
        - name: clickhouse
          image: clickhouse/clickhouse-server:24.8
          env:
            - name: CLICKHOUSE_DB
              value: default
            - name: CLICKHOUSE_USER
              valueFrom:
                secretKeyRef:
                  name: clickhouse-credentials
                  key: username
            - name: CLICKHOUSE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: clickhouse-credentials
                  key: password
          ports:
            - containerPort: 8123
            - containerPort: 9000
          readinessProbe:
            httpGet:
              path: /ping
              port: 8123
            initialDelaySeconds: 15
            periodSeconds: 5
          volumeMounts:
            - name: init
              mountPath: /docker-entrypoint-initdb.d
      volumes:
        - name: init
          configMap:
            name: clickhouse-init
---
apiVersion: v1
kind: Service
metadata:
  name: clickhouse
  namespace: data
spec:
  selector:
    app: clickhouse
  ports:
    - name: http
      port: 8123
      targetPort: 8123
    - name: native
      port: 9000
      targetPort: 9000
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spark-master
  namespace: spark
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spark-master
  template:
    metadata:
      labels:
        app: spark-master
    spec:
      containers:
        - name: spark-master
          image: bitnami/spark:3.5.1
          env:
            - name: SPARK_MODE
              value: master
            - name: SPARK_NO_DAEMONIZE
              value: "yes"
          ports:
            - containerPort: 7077
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: spark-master
  namespace: spark
spec:
  selector:
    app: spark-master
  ports:
    - name: spark
      port: 7077
      targetPort: 7077
    - name: web
      port: 8080
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spark-worker
  namespace: spark
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spark-worker
  template:
    metadata:
      labels:
        app: spark-worker
    spec:
      containers:
        - name: spark-worker
          image: bitnami/spark:3.5.1
          env:
            - name: SPARK_MODE
              value: worker
            - name: SPARK_MASTER_URL
              value: spark://spark-master.spark.svc.cluster.local:7077
            - name: SPARK_WORKER_MEMORY
              value: 2G
            - name: SPARK_WORKER_CORES
              value: "2"
            - name: SPARK_NO_DAEMONIZE
              value: "yes"
          ports:
            - containerPort: 8081
---
apiVersion: v1
kind: Secret
metadata:
  name: notebook-data-credentials
  namespace: jupyterhub
stringData:
  clickhouse_user: "${CLICKHOUSE_USER}"
  clickhouse_password: "${CLICKHOUSE_PASSWORD}"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iam-service
  namespace: clario360
spec:
  replicas: 1
  selector:
    matchLabels:
      app: iam-service
  template:
    metadata:
      labels:
        app: iam-service
    spec:
      containers:
        - name: iam-service
          image: clario360/iam-service:e2e
          imagePullPolicy: IfNotPresent
          env:
            - name: DATABASE_HOST
              value: postgresql.clario360.svc.cluster.local
            - name: DATABASE_PORT
              value: "5432"
            - name: DATABASE_USER
              value: clario
            - name: DATABASE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: platform-secrets
                  key: postgres-password
            - name: DATABASE_SSL_MODE
              value: disable
            - name: REDIS_HOST
              value: redis.clario360.svc.cluster.local
            - name: REDIS_PORT
              value: "6379"
            - name: KAFKA_BROKERS
              value: localhost:9092
            - name: AUTH_JWT_ISSUER
              value: http://localhost:${IAM_PORT}
            - name: CLARIO360_PUBLIC_URL
              value: http://localhost:${IAM_PORT}
            - name: NOTEBOOK_LOGIN_URL
              value: http://localhost:33000/login
            - name: JUPYTERHUB_OAUTH_CALLBACK_URL
              value: http://localhost:${JUPYTER_PORT}/hub/oauth_callback
            - name: JUPYTERHUB_API_URL
              value: http://hub.jupyterhub.svc.cluster.local:8081/hub/api
            - name: JUPYTERHUB_BASE_URL
              value: http://localhost:${JUPYTER_PORT}
            - name: JUPYTERHUB_ADMIN_TOKEN
              value: ${JUPYTERHUB_ADMIN_TOKEN}
            - name: JUPYTERHUB_OAUTH_CLIENT_SECRET
              value: ${JUPYTERHUB_CLIENT_SECRET}
            - name: OAUTH_ADDITIONAL_CLIENTS_JSON
              value: >-
                [{"client_id":"oidc-conformance","client_secret":"${CONFORMANCE_CLIENT_SECRET}","redirect_uris":["https://localhost:8443/test/a/clario360-oidcc/callback"],"scopes":["openid","profile","email"],"require_pkce":true}]
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: ""
          ports:
            - containerPort: 8081
            - containerPort: 9091
          readinessProbe:
            httpGet:
              path: /readyz
              port: 9091
            initialDelaySeconds: 10
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /healthz
              port: 9091
            initialDelaySeconds: 20
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: iam-service
  namespace: clario360
spec:
  selector:
    app: iam-service
  ports:
    - name: http
      port: 8081
      targetPort: 8081
    - name: admin
      port: 9091
      targetPort: 9091
EOF

wait_for_rollout clario360 postgresql
wait_for_rollout clario360 redis
wait_for_rollout data clickhouse
wait_for_rollout spark spark-master
wait_for_rollout spark spark-worker

log "Running platform_core migrations"
kubectl --context "kind-${CLUSTER_NAME}" -n clario360 delete job platform-core-migrate --ignore-not-found >/dev/null 2>&1 || true
kubectl --context "kind-${CLUSTER_NAME}" apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: platform-core-migrate
  namespace: clario360
spec:
  backoffLimit: 1
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrator
          image: clario360/migrator:e2e
          imagePullPolicy: IfNotPresent
          args: ["-direction", "up", "-db", "platform_core"]
          env:
            - name: DATABASE_HOST
              value: postgresql.clario360.svc.cluster.local
            - name: DATABASE_PORT
              value: "5432"
            - name: DATABASE_USER
              value: clario
            - name: DATABASE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: platform-secrets
                  key: postgres-password
            - name: DATABASE_SSL_MODE
              value: disable
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: ""
EOF
kubectl --context "kind-${CLUSTER_NAME}" -n clario360 wait --for=condition=complete job/platform-core-migrate --timeout=300s

log "Seeding notebook tenant"
kubectl --context "kind-${CLUSTER_NAME}" -n clario360 exec deployment/postgresql -- sh -lc "PGPASSWORD='${POSTGRES_PASSWORD}' psql -U clario -d platform_core -c \"INSERT INTO tenants (id, name, slug, status, subscription_tier, settings) VALUES ('${TEST_TENANT_ID}', 'Notebook E2E', 'notebook-e2e', 'active', 'enterprise', '{}'::jsonb) ON CONFLICT (slug) DO NOTHING;\""

log "Starting IAM service"
wait_for_rollout clario360 iam-service

log "Preparing JupyterHub chart dependencies"
helm dependency build "${ROOT_DIR}/deploy/jupyter/helm"

VALUES_FILE="$(render_values)"

log "Deploying JupyterHub"
helm upgrade --install clario360-jupyter "${ROOT_DIR}/deploy/jupyter/helm" \
  --namespace jupyterhub \
  --values "${ROOT_DIR}/deploy/jupyter/helm/values.yaml" \
  --values "${VALUES_FILE}" \
  --wait \
  --timeout 15m

log "Starting local port-forwards"
kubectl --context "kind-${CLUSTER_NAME}" -n clario360 port-forward service/iam-service "${IAM_PORT}:8081" >"${TEMP_DIR}/iam-port-forward.log" 2>&1 &
IAM_PORT_FORWARD_PID="$!"
kubectl --context "kind-${CLUSTER_NAME}" -n jupyterhub port-forward service/proxy-public "${JUPYTER_PORT}:80" >"${TEMP_DIR}/jupyter-port-forward.log" 2>&1 &
JUPYTER_PORT_FORWARD_PID="$!"
sleep 5

wait_for_http "http://localhost:${IAM_PORT}/.well-known/openid-configuration" 90
wait_for_http "http://localhost:${JUPYTER_PORT}/hub/health" 90

write_env_file
export CLUSTER_NAME
export IAM_BASE_URL="http://localhost:${IAM_PORT}"
export JUPYTER_BASE_URL="http://localhost:${JUPYTER_PORT}"
export TEST_TENANT_ID TEST_EMAIL TEST_PASSWORD

log "Running notebook smoke test"
python3 "${E2E_DIR}/smoke_test.py"

log "Harness complete"
echo "Environment file written to ${ENV_FILE}"
