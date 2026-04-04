# TS-005: WebSocket Connectivity Issues

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | TS-005                                     |
| **Title**        | WebSocket Connectivity Issues              |
| **Severity**     | P2 - High                                  |
| **Services**     | notification-service, api-gateway          |
| **Last Updated** | 2026-03-08                                 |
| **Author**       | Platform Engineering                       |
| **Review Cycle** | Quarterly                                  |

---

## Summary

This runbook covers the investigation and resolution of WebSocket connectivity issues in the Clario 360 platform. The notification-service manages WebSocket connections for real-time event delivery to the frontend. The api-gateway proxies WebSocket upgrade requests through to the notification-service. Disconnections may be caused by ingress/load balancer timeout settings, notification-service errors, heartbeat failures, client-side reconnection logic, or sticky session misconfiguration. Follow the diagnosis steps to isolate whether the issue is on the infrastructure, server, or client side.

---

## Symptoms

- Users report missing real-time notifications (new events only appear on page refresh).
- The frontend ConnectionStatusBanner shows a disconnected or reconnecting state.
- Browser developer console shows repeated WebSocket connection/disconnection cycles.
- Notification-service logs show high volumes of connection open/close events.
- Api-gateway logs show 502 or 504 errors on WebSocket upgrade requests.
- Grafana shows elevated WebSocket disconnection rates or reduced active connection counts.
- Client-side exponential backoff reaches maximum delay, indicating persistent failures.

---

## Diagnosis Steps

### Step 1: Check WebSocket Endpoint Health

```bash
# Check notification-service pod status
kubectl -n clario360 get pods -l app=notification-service -o wide

# Check notification-service health
kubectl -n clario360 exec deploy/notification-service -- curl -s http://localhost:8080/healthz
kubectl -n clario360 exec deploy/notification-service -- curl -s http://localhost:8080/readyz

# Check api-gateway health (WebSocket proxy layer)
kubectl -n clario360 exec deploy/api-gateway -- curl -s http://localhost:8080/healthz
kubectl -n clario360 exec deploy/api-gateway -- curl -s http://localhost:8080/readyz
```

```bash
# Test WebSocket upgrade from inside the cluster using curl
kubectl -n clario360 run ws-test --rm -it --image=curlimages/curl -- \
  curl -v -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Authorization: Bearer <valid-token>" \
  http://api-gateway.clario360.svc.cluster.local:8080/ws/notifications

# Expected: HTTP/1.1 101 Switching Protocols
# If you get 502/504, the issue is between api-gateway and notification-service.
# If you get 401/403, the issue is authentication.
```

```bash
# Test WebSocket upgrade directly to notification-service (bypass api-gateway)
kubectl -n clario360 run ws-test-direct --rm -it --image=curlimages/curl -- \
  curl -v -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Authorization: Bearer <valid-token>" \
  http://notification-service.clario360.svc.cluster.local:8080/ws

# If this succeeds but the api-gateway path fails, the issue is in the proxy layer.
```

### Step 2: Check Nginx / Ingress Timeout Settings

```bash
# Check ingress resource configuration
kubectl -n clario360 get ingress -o yaml

# Look specifically for WebSocket-related annotations
kubectl -n clario360 get ingress -o yaml | grep -i -E 'timeout|websocket|upgrade|proxy-read|proxy-send|proxy-connect'
```

Key annotations to verify on the ingress:

```
nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
nginx.ingress.kubernetes.io/proxy-connect-timeout: "60"
nginx.ingress.kubernetes.io/upstream-hash-by: "$remote_addr"
nginx.ingress.kubernetes.io/configuration-snippet: |
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
```

```bash
# Check nginx-ingress-controller configuration
kubectl -n ingress-nginx get configmap nginx-configuration -o yaml | grep -i -E 'timeout|websocket|keepalive|upgrade'

# Check nginx-ingress-controller logs for WebSocket-related errors
kubectl -n ingress-nginx logs deploy/nginx-ingress-controller --tail=200 | grep -i -E 'websocket|upgrade|502|504|timeout'
```

```bash
# If using a cloud load balancer, check its timeout settings
# GCP:
kubectl -n clario360 get svc -o yaml | grep -A 20 'annotations' | grep -i -E 'timeout|idle|backend'

# The idle timeout on the load balancer must be >= the WebSocket ping interval.
# Default GCP idle timeout is 600 seconds (10 minutes).
```

### Step 3: Check Notification-Service Logs for Connection Errors

```bash
# Check for WebSocket connection errors
kubectl -n clario360 logs deploy/notification-service --tail=500 | grep -i -E 'websocket|ws|connection|disconnect|close|error|upgrade'

# Check for patterns of rapid connect/disconnect
kubectl -n clario360 logs deploy/notification-service --tail=1000 | grep -i -E 'connect|disconnect' | head -50

# Count connections and disconnections in the last 5 minutes
echo "Connections:"
kubectl -n clario360 logs deploy/notification-service --since=5m | grep -i -c 'connect'
echo "Disconnections:"
kubectl -n clario360 logs deploy/notification-service --since=5m | grep -i -c 'disconnect\|close'

# Check for goroutine leaks (too many concurrent connections)
kubectl -n clario360 exec deploy/notification-service -- curl -s http://localhost:8080/metrics | grep -E 'goroutines|websocket_connections'

# Check for memory pressure (goroutine leaks cause memory growth)
kubectl -n clario360 top pod -l app=notification-service
```

```bash
# Check api-gateway logs for WebSocket proxy errors
kubectl -n clario360 logs deploy/api-gateway --tail=500 | grep -i -E 'websocket|ws|proxy.*error|upgrade.*fail|502|504'
```

### Step 4: Check Client-Side Reconnection Behavior

The frontend uses exponential backoff for WebSocket reconnections. Check the current state:

```bash
# Check api-gateway access logs for repeated WebSocket upgrade attempts from the same client
kubectl -n clario360 logs deploy/api-gateway --tail=1000 | grep -i 'upgrade' | awk '{print $1}' | sort | uniq -c | sort -rn | head -20

# Check notification-service for rapid reconnection patterns (same user reconnecting frequently)
kubectl -n clario360 logs deploy/notification-service --tail=1000 | grep -i 'connect' | grep -oP 'user[=:]\S+' | sort | uniq -c | sort -rn | head -20
```

Expected reconnection behavior:
- Initial retry: 1 second
- Backoff multiplier: 2x
- Maximum delay: 30 seconds
- The client should not reconnect more than ~2-3 times per minute at steady state.

If a single client is reconnecting every 1-2 seconds continuously, the connection is being immediately closed after establishment. Check the notification-service logs for the specific user.

### Step 5: Verify Load Balancer Sticky Sessions

WebSocket connections must remain on the same backend pod for the duration of the connection. If the load balancer routes the upgrade request and subsequent frames to different pods, the connection will fail.

```bash
# Check if the service uses sessionAffinity
kubectl -n clario360 get svc notification-service -o yaml | grep -A 5 sessionAffinity

# Check ingress upstream hash configuration
kubectl -n clario360 get ingress -o yaml | grep -i -E 'affinity|sticky|hash|session'

# Verify the number of notification-service replicas (sticky sessions matter with >1 replica)
kubectl -n clario360 get deploy notification-service -o jsonpath='{.spec.replicas}'
echo ""

# Check if connections are distributed evenly across pods
for pod in $(kubectl -n clario360 get pods -l app=notification-service -o name); do
  echo "=== $pod ==="
  kubectl -n clario360 exec $pod -- curl -s http://localhost:8080/metrics | grep 'websocket_active_connections'
done
```

### Step 6: Check Heartbeat / Ping-Pong Configuration

```bash
# Check notification-service configuration for ping/pong settings
kubectl -n clario360 get configmap notification-service-config -o yaml | grep -i -E 'ping|pong|heartbeat|interval|timeout|keepalive'

# Check api-gateway WebSocket proxy settings
kubectl -n clario360 get configmap api-gateway-config -o yaml | grep -i -E 'websocket|ping|pong|heartbeat|timeout'
```

The WebSocket ping interval must be shorter than all timeout values in the chain:
- Notification-service ping interval (e.g., 30s)
- Api-gateway proxy read timeout
- Ingress proxy read timeout
- Load balancer idle timeout

If any timeout in the chain is shorter than the ping interval, idle connections will be terminated.

### Step 7: Check Resource Limits and Connection Counts

```bash
# Check notification-service resource usage
kubectl -n clario360 top pod -l app=notification-service

# Check notification-service resource limits
kubectl -n clario360 describe deploy/notification-service | grep -A 10 'Resources'

# Check active WebSocket connection count via metrics
kubectl -n clario360 port-forward svc/notification-service 8081:8080
curl -s http://localhost:8081/metrics | grep -E 'websocket|connections|goroutines'

# Check file descriptor limits (each WebSocket connection uses a file descriptor)
kubectl -n clario360 exec deploy/notification-service -- sh -c 'cat /proc/1/limits | grep "Max open files"'

# Check current file descriptor usage
kubectl -n clario360 exec deploy/notification-service -- sh -c 'ls /proc/1/fd | wc -l'

# Check system-wide connection limits
kubectl -n clario360 exec deploy/notification-service -- sh -c 'cat /proc/sys/net/core/somaxconn'
```

---

## Resolution Steps

### Resolution: Adjust Ingress/Load Balancer Timeouts

```bash
# Update ingress annotations for WebSocket support
kubectl -n clario360 annotate ingress clario360-ingress --overwrite \
  nginx.ingress.kubernetes.io/proxy-read-timeout="3600" \
  nginx.ingress.kubernetes.io/proxy-send-timeout="3600" \
  nginx.ingress.kubernetes.io/proxy-connect-timeout="60" \
  nginx.ingress.kubernetes.io/connection-proxy-header="keep-alive" \
  nginx.ingress.kubernetes.io/upstream-hash-by="\$remote_addr"

# Add WebSocket-specific configuration snippet
kubectl -n clario360 get ingress clario360-ingress -o yaml > /tmp/ingress.yaml
# Edit to add:
#   nginx.ingress.kubernetes.io/configuration-snippet: |
#     proxy_set_header Upgrade $http_upgrade;
#     proxy_set_header Connection "upgrade";
#     proxy_http_version 1.1;
kubectl apply -f /tmp/ingress.yaml

# Verify the nginx configuration was updated
kubectl -n ingress-nginx exec deploy/nginx-ingress-controller -- cat /etc/nginx/nginx.conf | grep -A 5 'websocket\|upgrade'
```

```bash
# If using GCP BackendConfig for load balancer timeout
kubectl -n clario360 apply -f - <<EOF
apiVersion: cloud.google.com/v1
kind: BackendConfig
metadata:
  name: clario360-websocket-config
  namespace: clario360
spec:
  timeoutSec: 3600
  connectionDraining:
    drainingTimeoutSec: 60
EOF

# Annotate the service to use the BackendConfig
kubectl -n clario360 annotate svc notification-service \
  cloud.google.com/backend-config='{"default":"clario360-websocket-config"}'
```

### Resolution: Fix Heartbeat Configuration

```bash
# Update notification-service heartbeat settings
kubectl -n clario360 patch configmap notification-service-config --type='merge' -p='{
  "data": {
    "WS_PING_INTERVAL": "25s",
    "WS_PONG_TIMEOUT": "10s",
    "WS_WRITE_TIMEOUT": "10s",
    "WS_MAX_MESSAGE_SIZE": "65536"
  }
}'

# Restart notification-service to apply
kubectl -n clario360 rollout restart deploy/notification-service
kubectl -n clario360 rollout status deploy/notification-service
```

The ping interval (25s) must be less than:
- Ingress proxy-read-timeout (3600s) -- OK
- Load balancer idle timeout (600s) -- OK
- Api-gateway proxy timeout -- verify this is >= 600s

```bash
# Update api-gateway WebSocket proxy timeout if needed
kubectl -n clario360 patch configmap api-gateway-config --type='merge' -p='{
  "data": {
    "WS_PROXY_READ_TIMEOUT": "3600s",
    "WS_PROXY_WRITE_TIMEOUT": "60s"
  }
}'

kubectl -n clario360 rollout restart deploy/api-gateway
kubectl -n clario360 rollout status deploy/api-gateway
```

### Resolution: Enable Sticky Sessions

```bash
# Option 1: Enable sessionAffinity on the Kubernetes service
kubectl -n clario360 patch svc notification-service -p '{"spec":{"sessionAffinity":"ClientIP","sessionAffinityConfig":{"clientIP":{"timeoutSeconds":3600}}}}'

# Option 2: Use cookie-based affinity via ingress annotations
kubectl -n clario360 annotate ingress clario360-ingress --overwrite \
  nginx.ingress.kubernetes.io/affinity="cookie" \
  nginx.ingress.kubernetes.io/affinity-mode="persistent" \
  nginx.ingress.kubernetes.io/session-cookie-name="CLARIO_WS_AFFINITY" \
  nginx.ingress.kubernetes.io/session-cookie-expires="3600" \
  nginx.ingress.kubernetes.io/session-cookie-max-age="3600" \
  nginx.ingress.kubernetes.io/session-cookie-samesite="Strict" \
  nginx.ingress.kubernetes.io/session-cookie-secure="true"
```

### Resolution: Increase Connection Limits

```bash
# Increase notification-service resource limits to handle more connections
kubectl -n clario360 patch deploy/notification-service --type='json' -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "250m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value": "1000m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "256Mi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "1Gi"}
]'

# Increase file descriptor limits via security context
kubectl -n clario360 patch deploy/notification-service --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/securityContext/capabilities", "value": {"add": ["SYS_RESOURCE"]}}
]'

# Or set ulimits via init container
kubectl -n clario360 patch deploy/notification-service --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/initContainers", "value": [
    {
      "name": "sysctl",
      "image": "busybox",
      "command": ["sh", "-c", "sysctl -w net.core.somaxconn=65535"],
      "securityContext": {"privileged": true}
    }
  ]}
]'

# Update the max connections configuration
kubectl -n clario360 patch configmap notification-service-config --type='merge' -p='{
  "data": {
    "WS_MAX_CONNECTIONS": "10000",
    "WS_MAX_CONNECTIONS_PER_USER": "5"
  }
}'

kubectl -n clario360 rollout restart deploy/notification-service
kubectl -n clario360 rollout status deploy/notification-service
```

### Resolution: Scale Notification Service

```bash
# Scale horizontally to distribute WebSocket connections across more pods
kubectl -n clario360 scale deploy/notification-service --replicas=4

# IMPORTANT: When scaling notification-service, ensure sticky sessions are configured
# (see sticky sessions resolution above) and that Redis pub/sub is used for
# cross-pod message delivery.

# Verify all replicas are ready
kubectl -n clario360 rollout status deploy/notification-service

# Check connection distribution across pods
for pod in $(kubectl -n clario360 get pods -l app=notification-service -o name); do
  echo "=== $pod ==="
  kubectl -n clario360 exec $pod -- curl -s http://localhost:8080/metrics | grep 'websocket_active_connections'
done
```

### Resolution: Fix Api-Gateway WebSocket Proxy

```bash
# If the api-gateway is not properly proxying WebSocket upgrades, check its configuration
kubectl -n clario360 get configmap api-gateway-config -o yaml

# Ensure WebSocket proxy is enabled
kubectl -n clario360 patch configmap api-gateway-config --type='merge' -p='{
  "data": {
    "WS_PROXY_ENABLED": "true",
    "WS_PROXY_TARGET": "http://notification-service.clario360.svc.cluster.local:8080",
    "WS_PROXY_PATH": "/ws"
  }
}'

kubectl -n clario360 rollout restart deploy/api-gateway
kubectl -n clario360 rollout status deploy/api-gateway
```

---

## Verification

After applying the resolution, verify the fix:

```bash
# 1. Check all service health endpoints
kubectl -n clario360 exec deploy/notification-service -- curl -s http://localhost:8080/healthz
kubectl -n clario360 exec deploy/notification-service -- curl -s http://localhost:8080/readyz
kubectl -n clario360 exec deploy/api-gateway -- curl -s http://localhost:8080/healthz

# 2. Test WebSocket upgrade through the full chain
kubectl -n clario360 run ws-verify --rm -it --image=curlimages/curl -- \
  curl -v -N --max-time 10 \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Authorization: Bearer <valid-token>" \
  http://api-gateway.clario360.svc.cluster.local:8080/ws/notifications
# Expected: HTTP/1.1 101 Switching Protocols

# 3. Check active WebSocket connection metrics
kubectl -n clario360 port-forward svc/notification-service 8081:8080
curl -s http://localhost:8081/metrics | grep -E 'websocket_active_connections|websocket_total_connections|websocket_errors'

# 4. Verify no connection churn in logs
echo "Connections in last 5 minutes:"
kubectl -n clario360 logs deploy/notification-service --since=5m | grep -i -c 'connected'
echo "Disconnections in last 5 minutes:"
kubectl -n clario360 logs deploy/notification-service --since=5m | grep -i -c 'disconnect'
# Disconnection count should be significantly less than connection count

# 5. Verify ingress configuration was applied
kubectl -n clario360 get ingress clario360-ingress -o yaml | grep -E 'timeout|websocket|upgrade|affinity'

# 6. Check notification-service pod resource usage is healthy
kubectl -n clario360 top pod -l app=notification-service

# 7. Verify sticky sessions are working (multiple requests hit the same pod)
for i in $(seq 1 5); do
  kubectl -n clario360 run sticky-test-$i --rm -it --image=curlimages/curl -- \
    curl -s http://notification-service.clario360.svc.cluster.local:8080/healthz -D - | grep -i 'pod\|server'
done

# 8. Check connection distribution across replicas
for pod in $(kubectl -n clario360 get pods -l app=notification-service -o name); do
  echo "=== $pod ==="
  kubectl -n clario360 exec $pod -- curl -s http://localhost:8080/metrics | grep 'websocket_active_connections'
done

# 9. Verify no errors in recent logs
kubectl -n clario360 logs deploy/notification-service --since=5m | grep -i error | wc -l
kubectl -n clario360 logs deploy/api-gateway --since=5m | grep -i -E 'websocket.*error|502|504' | wc -l
```

---

## Related Links

- [TS-001: API Latency Investigation](./TS-001-slow-api-responses.md)
- [TS-002: Data Pipeline Failure Investigation](./TS-002-failed-pipelines.md)
- [TS-003: Kafka Event Loss Investigation](./TS-003-missing-events.md)
- [TS-004: Authentication Issue Debugging](./TS-004-auth-failures.md)
- Grafana WebSocket Dashboard: `/d/websocket-monitoring/websocket-connections`
- Grafana Notification Service Dashboard: `/d/notification-service/notification-service`
- WebSocket Protocol RFC 6455: https://datatracker.ietf.org/doc/html/rfc6455
- Nginx WebSocket Proxying: https://nginx.org/en/docs/http/websocket.html
