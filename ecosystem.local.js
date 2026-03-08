const crypto = require("crypto");
const fs = require("fs");
const path = require("path");

const repoRoot = __dirname;
const backendDir = path.join(repoRoot, "backend");
const frontendDir = path.join(repoRoot, "frontend");
const binDir = path.join(repoRoot, ".dev-bin");
const goCacheDir = path.join(repoRoot, ".cache", "go-build");
const secretsDir = path.join(repoRoot, ".dev-secrets");

fs.mkdirSync(secretsDir, { recursive: true });
fs.mkdirSync(goCacheDir, { recursive: true });

function readTrimmed(filePath) {
  return fs.readFileSync(filePath, "utf8").trim();
}

function ensureBase64Secret(name) {
  const filePath = path.join(secretsDir, name);
  if (!fs.existsSync(filePath)) {
    fs.writeFileSync(filePath, crypto.randomBytes(32).toString("base64"));
  }
  return readTrimmed(filePath);
}

function binaryOrGo(serviceName) {
  const binary = path.join(binDir, serviceName);
  if (fs.existsSync(binary)) {
    return {
      script: binary,
      interpreter: "none",
    };
  }

  return {
    script: "go",
    args: `run ./cmd/${serviceName}`,
    interpreter: "none",
  };
}

function serviceApp(serviceName, env) {
  return {
    name: `clario360-${serviceName}`,
    cwd: backendDir,
    ...binaryOrGo(serviceName),
    env: {
      ...sharedEnv,
      ...env,
    },
  };
}

const jwtPrivateKey = readTrimmed(path.join(secretsDir, "jwt-private.pem"));
const jwtPublicKey = readTrimmed(path.join(secretsDir, "jwt-public.pem"));
const dataEncryptionKey = ensureBase64Secret("data-encryption.key");
const fileEncryptionKey = ensureBase64Secret("file-encryption.key");
const platformEncryptionKey = ensureBase64Secret("encryption.key");
const webhookHmacSecretPath = path.join(secretsDir, "webhook-hmac.key");
if (!fs.existsSync(webhookHmacSecretPath)) {
  fs.writeFileSync(webhookHmacSecretPath, crypto.randomBytes(32).toString("hex"));
}
const webhookHmacSecret = readTrimmed(webhookHmacSecretPath);

const sharedEnv = {
  ENVIRONMENT: "development",
  OBSERVABILITY_LOG_LEVEL: "info",
  OBSERVABILITY_LOG_FORMAT: "json",
  OBSERVABILITY_OTLP_ENDPOINT: "",

  DATABASE_HOST: "localhost",
  DATABASE_PORT: "5432",
  DATABASE_USER: "clario",
  DATABASE_PASSWORD: "clario_dev_pass",
  DATABASE_NAME: "clario360",
  DATABASE_SSL_MODE: "disable",
  DATABASE_MAX_OPEN_CONNS: "8",
  DATABASE_MAX_IDLE_CONNS: "2",
  DATABASE_CONN_MAX_LIFETIME: "5m",

  REDIS_HOST: "127.0.0.1",
  REDIS_PORT: "6379",
  REDIS_PASSWORD: "",
  REDIS_DB: "0",

  KAFKA_BROKERS: "localhost:9094",
  KAFKA_GROUP_ID: "clario360",
  KAFKA_AUTO_OFFSET_RESET: "earliest",

  GOWORK: "off",
  GOCACHE: goCacheDir,

  AUTH_RSA_PRIVATE_KEY_PEM: jwtPrivateKey,
  AUTH_RSA_PUBLIC_KEY_PEM: jwtPublicKey,
  AUTH_JWT_ISSUER: "clario360",
  AUTH_JWT_ACCESS_TOKEN_TTL: "15m",
  AUTH_JWT_REFRESH_TOKEN_TTL: "168h",
  AUTH_BCRYPT_COST: "12",

  ENCRYPTION_KEY: platformEncryptionKey,

  MINIO_ENDPOINT: "localhost:9000",
  MINIO_ACCESS_KEY: "clario_minio",
  MINIO_SECRET_KEY: "clario_minio_secret",
  MINIO_USE_SSL: "false",
  MINIO_BUCKET: "clario360",
};

module.exports = {
  apps: [
    serviceApp("iam-service", {}),
    serviceApp("audit-service", {
      AUDIT_HTTP_PORT: "8084",
      AUDIT_DB_MIN_CONNS: "1",
      AUDIT_DB_MAX_CONNS: "4",
      AUDIT_MINIO_ENDPOINT: "localhost:9000",
      AUDIT_MINIO_ACCESS_KEY: "clario_minio",
      AUDIT_MINIO_SECRET_KEY: "clario_minio_secret",
      AUDIT_MINIO_BUCKET: "audit-exports",
    }),
    serviceApp("notification-service", {
      NOTIF_HTTP_PORT: "8090",
      NOTIF_DB_MIN_CONNS: "1",
      NOTIF_DB_MAX_CONNS: "4",
      NOTIF_EMAIL_PROVIDER: "smtp",
      NOTIF_SMTP_HOST: "localhost",
      NOTIF_SMTP_PORT: "1025",
      NOTIF_SMTP_TLS_ENABLED: "false",
      NOTIF_WEBHOOK_HMAC_SECRET: webhookHmacSecret,
      NOTIF_WS_ALLOWED_ORIGINS: "http://localhost:3000",
      NOTIF_IAM_SERVICE_URL: "http://localhost:8081",
      NOTIF_DATA_SERVICE_URL: "http://localhost:8086",
      NOTIF_ACTA_SERVICE_URL: "http://localhost:8087",
      NOTIF_CYBER_SERVICE_URL: "http://localhost:8085",
      NOTIF_ENVIRONMENT: "development",
    }),
    serviceApp("workflow-engine", {
      WF_HTTP_PORT: "8083",
      WF_SERVICE_URLS: "notification=http://localhost:8090,cyber=http://localhost:8085",
    }),
    serviceApp("cyber-service", {
      CYBER_HTTP_PORT: "8085",
      CYBER_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/cyber_db?sslmode=disable",
      CYBER_DB_MIN_CONNS: "1",
      CYBER_DB_MAX_CONNS: "4",
      CYBER_REDIS_URL: "redis://127.0.0.1:6379/1",
      CYBER_KAFKA_BROKERS: "localhost:9094",
      CYBER_KAFKA_GROUP_ID: "cyber-service",
      CYBER_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
    }),
    serviceApp("data-service", {
      DATA_HTTP_PORT: "8086",
      DATA_ADMIN_PORT: "9086",
      DATA_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/data_db?sslmode=disable",
      DATA_DB_MIN_CONNS: "1",
      DATA_DB_MAX_CONNS: "4",
      DATA_REDIS_URL: "redis://127.0.0.1:6379/2",
      DATA_KAFKA_BROKERS: "localhost:9094",
      DATA_KAFKA_GROUP_ID: "data-service",
      DATA_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
      DATA_ENCRYPTION_KEY: dataEncryptionKey,
      DATA_MINIO_ENDPOINT: "localhost:9000",
      DATA_MINIO_ACCESS_KEY: "clario_minio",
      DATA_MINIO_SECRET_KEY: "clario_minio_secret",
    }),
    serviceApp("acta-service", {
      ACTA_HTTP_PORT: "8087",
      ACTA_ADMIN_PORT: "9087",
      ACTA_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/acta_db?sslmode=disable",
      ACTA_DB_MIN_CONNS: "1",
      ACTA_DB_MAX_CONNS: "4",
      ACTA_REDIS_ADDR: "127.0.0.1:6379",
      ACTA_KAFKA_BROKERS: "localhost:9094",
      ACTA_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
      ACTA_SEED_DEMO_DATA: "false",
    }),
    serviceApp("lex-service", {
      LEX_HTTP_PORT: "8088",
      LEX_ADMIN_PORT: "9088",
      LEX_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/lex_db?sslmode=disable",
      LEX_DB_MIN_CONNS: "1",
      LEX_DB_MAX_CONNS: "4",
      LEX_REDIS_ADDR: "127.0.0.1:6379",
      LEX_KAFKA_BROKERS: "localhost:9094",
      LEX_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
      LEX_SEED_DEMO_DATA: "false",
    }),
    serviceApp("visus-service", {
      VISUS_HTTP_PORT: "8089",
      VISUS_ADMIN_PORT: "9089",
      VISUS_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/visus_db?sslmode=disable",
      VISUS_DB_MIN_CONNS: "1",
      VISUS_DB_MAX_CONNS: "4",
      VISUS_REDIS_ADDR: "127.0.0.1:6379",
      VISUS_KAFKA_BROKERS: "localhost:9094",
      VISUS_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
      VISUS_JWT_PRIVATE_KEY_PATH: path.join(secretsDir, "jwt-private.pem"),
      VISUS_SUITE_CYBER_URL: "http://localhost:8085",
      VISUS_SUITE_DATA_URL: "http://localhost:8086",
      VISUS_SUITE_ACTA_URL: "http://localhost:8087",
      VISUS_SUITE_LEX_URL: "http://localhost:8088",
      VISUS_SEED_DEMO_DATA: "false",
    }),
    serviceApp("file-service", {
      FILE_HTTP_PORT: "8091",
      FILE_DB_URL: "postgres://clario:clario_dev_pass@localhost:5432/platform_core?sslmode=disable",
      FILE_DB_MIN_CONNS: "1",
      FILE_DB_MAX_CONNS: "4",
      FILE_REDIS_URL: "127.0.0.1:6379",
      FILE_KAFKA_BROKERS: "localhost:9094",
      FILE_KAFKA_GROUP_ID: "file-service",
      FILE_JWT_PUBLIC_KEY_PATH: path.join(secretsDir, "jwt-public.pem"),
      FILE_MINIO_ENDPOINT: "localhost:9000",
      FILE_MINIO_ACCESS_KEY: "clario_minio",
      FILE_MINIO_SECRET_KEY: "clario_minio_secret",
      FILE_MINIO_USE_SSL: "false",
      FILE_MINIO_BUCKET_PREFIX: "clario360",
      FILE_ENCRYPTION_MASTER_KEY: fileEncryptionKey,
      FILE_CLAMAV_ADDRESS: "localhost:3310",
      FILE_ENVIRONMENT: "development",
    }),
    serviceApp("api-gateway", {
      GW_HTTP_PORT: "8080",
      GW_ADMIN_PORT: "9080",
      GW_ENVIRONMENT: "development",
      GW_CORS_ALLOWED_ORIGINS: "http://localhost:3000,http://localhost:3001",
      GW_READ_TIMEOUT_SEC: "15",
      GW_WRITE_TIMEOUT_SEC: "60",
      GW_PROXY_TIMEOUT_SEC: "30",
      GW_SVC_URL_IAM: "http://localhost:8081",
      GW_SVC_URL_AUDIT: "http://localhost:8084",
      GW_SVC_URL_WORKFLOW: "http://localhost:8083",
      GW_SVC_URL_NOTIFICATION: "http://localhost:8090",
      GW_SVC_URL_FILE: "http://localhost:8091",
      GW_SVC_URL_CYBER: "http://localhost:8085",
      GW_SVC_URL_DATA: "http://localhost:8086",
      GW_SVC_URL_ACTA: "http://localhost:8087",
      GW_SVC_URL_LEX: "http://localhost:8088",
      GW_SVC_URL_VISUS: "http://localhost:8089",
    }),
    {
      name: "clario360-frontend",
      cwd: frontendDir,
      script: "npm",
      args: "run dev -- -H 0.0.0.0 -p 3000",
      interpreter: "none",
      env: {
        NODE_ENV: "development",
        NEXT_PUBLIC_API_URL: "http://localhost:8080",
        NEXT_PUBLIC_APP_NAME: "Clario 360",
        NEXT_PUBLIC_APP_URL: "http://localhost:3000",
        NEXT_PUBLIC_APP_VERSION: "0.1.0",
        AUTH_COOKIE_NAME: "clario360",
        AUTH_COOKIE_SECURE: "false",
        AUTH_COOKIE_DOMAIN: "localhost",
        AUTH_COOKIE_SAMESITE: "strict",
        AUTH_ACCESS_TOKEN_MAX_AGE: "900",
        AUTH_REFRESH_TOKEN_MAX_AGE: "604800",
      },
    },
  ],
};
