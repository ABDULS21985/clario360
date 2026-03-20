# TS-010: Cross-Suite Event Integration Problems

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | TS-010                                                                |
| **Title**          | Cross-Suite Event Integration Not Flowing                             |
| **Severity**       | P2 -- High                                                            |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | api-gateway, iam-service, audit-service, workflow-engine, notification-service, cyber-service, data-service, acta-service, lex-service, visus-service |
| **Namespace**      | clario360, kafka                                                      |
| **Escalation**     | Platform Engineering Lead -> VP Engineering -> CTO                    |
| **SLA**            | Acknowledge within 10 minutes, resolve within 1 hour                  |

---

## Summary

This runbook addresses problems with cross-suite event integration in the Clario360 platform. Services communicate asynchronously through Kafka topics for cross-suite events (e.g., a cyber-service threat detection triggers a workflow in workflow-engine and a notification via notification-service). When events stop flowing, downstream services do not receive updates, causing stale data, missed notifications, broken workflows, and compliance gaps. Common causes include missing Kafka topics, producer failures, consumer lag, schema incompatibility, ACL misconfigurations, Kafka broker issues, and serialization errors.

---

## Symptoms

- Downstream services not reacting to events from upstream services (e.g., workflows not triggered by cyber events).
- Notification service not sending alerts for events published by other services.
- Audit service missing cross-suite audit entries.
- Consumer lag growing continuously on cross-suite topics (visible in Kafka metrics or Grafana).
- Producer services logging `kafka: message too large`, `topic not found`, `authorization failed`, or `serialization error`.
- Consumer services logging `deserialization error`, `unknown schema`, or `consumer group rebalancing` repeatedly.
- Prometheus alerts `kafka_consumer_lag_high` or `kafka_producer_error_rate_high` firing.
- Cross-suite dashboards in visus-service showing stale or missing data.

---

## Impact Assessment

| Event Flow                        | Impact When Broken                                                |
|-----------------------------------|-------------------------------------------------------------------|
| cyber -> workflow-engine          | Security incidents do not trigger automated response workflows    |
| cyber -> notification-service     | Security alerts not sent to administrators                        |
| acta -> audit-service             | Document compliance events not recorded in audit trail            |
| lex -> notification-service       | Regulatory deadline notifications not delivered                   |
| data -> visus-service             | Dashboards show stale data; reporting inaccurate                  |
| workflow-engine -> notification   | Task assignments and approvals not communicated to users          |
| iam-service -> audit-service      | Authentication and authorization events not logged                |
| Any service -> audit-service      | Compliance audit trail has gaps                                   |

---

## Prerequisites

- `kubectl` configured with cluster access and `clario360` and `kafka` namespace permissions.
- Access to Kafka CLI tools (`kafka-topics.sh`, `kafka-consumer-groups.sh`, `kafka-console-consumer.sh`) or ability to exec into a Kafka broker pod.
- Access to Grafana dashboards for Kafka metrics.
- Understanding of the Clario360 event schema format (JSON or Avro).

---

## Diagnosis Steps

### Step 1: Verify Kafka Broker Health

```bash
# Check Kafka broker pods
kubectl get pods -n kafka -l app=kafka -o wide

# Check Kafka broker logs for errors
kubectl logs -n kafka -l app=kafka --tail=100 --timestamps | grep -i -E "error|exception|failed|shutdown"

# Check Kafka controller status
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-metadata.sh --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log --cluster-id $(kafka-storage.sh random-uuid) 2>/dev/null || \
  echo "Check broker logs for controller status"
```

### Step 2: List All Cross-Suite Event Topics

```bash
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 --list | grep -E "clario360|cross-suite|events"
```

Expected topics (adjust to your naming convention):

```
clario360.events.cyber
clario360.events.workflow
clario360.events.notification
clario360.events.audit
clario360.events.iam
clario360.events.data
clario360.events.acta
clario360.events.lex
clario360.events.visus
clario360.events.cross-suite
```

### Step 3: Check Topic Details

```bash
# Check a specific topic's configuration and partitions
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --describe --topic clario360.events.<TOPIC_NAME>
```

Check all event topics:

```bash
for TOPIC in cyber workflow notification audit iam data acta lex visus cross-suite; do
  echo "--- clario360.events.$TOPIC ---"
  kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
    kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
    --describe --topic clario360.events.$TOPIC 2>&1
  echo ""
done
```

### Step 4: Check Consumer Group Status and Lag

```bash
# List consumer groups
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 --list | grep clario360

# Check lag for a specific consumer group
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --describe --group clario360-<SERVICE>-consumer
```

Check lag for all consumer groups:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "=== $SERVICE ==="
  kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
    kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
    --describe --group clario360-$SERVICE-consumer 2>&1
  echo ""
done
```

Look for:
- `LAG` column showing high or growing numbers.
- `CONSUMER-ID` column empty (no active consumer).
- `STATE` showing `Dead` or `Empty` (consumer group has no members).

### Step 5: Check Producer Service Logs for Publish Errors

```bash
# Check the suspected producer service
kubectl logs -n clario360 -l app=<PRODUCER_SERVICE> --tail=300 --timestamps | \
  grep -i -E "kafka|publish|produce|event.*error|event.*fail|topic|broker|serializ"
```

Check all services for Kafka producer errors:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE (producer) ---"
  kubectl logs -n clario360 -l app=$SERVICE --tail=100 --timestamps | \
    grep -i -E "kafka.*error|publish.*fail|produce.*error|topic not found|authorization|serializ" | tail -3
  echo ""
done
```

### Step 6: Check Consumer Service Logs for Consumption Errors

```bash
# Check the suspected consumer service
kubectl logs -n clario360 -l app=<CONSUMER_SERVICE> --tail=300 --timestamps | \
  grep -i -E "kafka|consume|subscribe|deserializ|schema|offset|rebalance|partition"
```

Check all services for Kafka consumer errors:

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE (consumer) ---"
  kubectl logs -n clario360 -l app=$SERVICE --tail=100 --timestamps | \
    grep -i -E "consume.*error|deserializ|unknown.*schema|rebalance|partition.*revoked|offset" | tail -3
  echo ""
done
```

### Step 7: Test Producing and Consuming a Test Event

```bash
# Produce a test event to verify end-to-end flow
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-console-producer.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --topic clario360.events.cross-suite <<'EOF'
{"event_type":"test.ping","source":"ts-010-runbook","timestamp":"2026-03-08T00:00:00Z","data":{"message":"connectivity test"}}
EOF

# Consume the test event to verify it was written
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-console-consumer.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --topic clario360.events.cross-suite \
  --from-beginning --max-messages 5 --timeout-ms 10000
```

### Step 8: Verify Event Schema Compatibility

```bash
# Check if a schema registry is running
kubectl get pods -n kafka -l app=schema-registry -o wide

# If schema registry exists, check registered schemas
kubectl port-forward -n kafka svc/schema-registry 8081:8081 &
PF_PID=$!
sleep 2

# List all registered subjects
curl -s http://localhost:8081/subjects | jq .

# Get the latest schema version for a specific topic
curl -s http://localhost:8081/subjects/clario360.events.<TOPIC_NAME>-value/versions/latest | jq .

# Check compatibility of a new schema
curl -s http://localhost:8081/compatibility/subjects/clario360.events.<TOPIC_NAME>-value/versions/latest \
  -H "Content-Type: application/json" \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"event_type\",\"type\":\"string\"}]}"}' | jq .

kill $PF_PID
```

### Step 9: Check Kafka ACLs on Cross-Suite Topics

```bash
# List all ACLs
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --list

# Check ACLs for a specific topic
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --list --topic clario360.events.<TOPIC_NAME>

# Check ACLs for a specific consumer group
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --list --group clario360-<SERVICE>-consumer
```

### Step 10: Check Kafka Broker Metrics

```bash
kubectl port-forward -n kafka svc/kafka 9090:9090 &
PF_PID=$!
sleep 2

# Check broker metrics if JMX/Prometheus exporter is enabled
curl -s http://localhost:9090/metrics 2>/dev/null | grep -E "kafka_server_BrokerTopic|kafka_server_Replica|UnderReplicatedPartitions|OfflinePartitionsCount" | head -20

kill $PF_PID
```

### Step 11: Check Service Kafka Configuration

```bash
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "--- $SERVICE ---"
  kubectl get deployment $SERVICE -n clario360 -o jsonpath='{.spec.template.spec.containers[0].env}' 2>/dev/null | \
    jq -r '.[] | select(.name | test("KAFKA|BROKER|TOPIC|CONSUMER|PRODUCER|EVENT"; "i")) | "\(.name)=\(.value // .valueFrom)"' 2>/dev/null
  echo ""
done
```

---

## Resolution Steps

### Resolution A: Create Missing Kafka Topics

If a required topic does not exist:

```bash
# Create a missing cross-suite event topic
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --create \
  --topic clario360.events.<TOPIC_NAME> \
  --partitions 6 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config cleanup.policy=delete \
  --config max.message.bytes=1048576

# Verify the topic was created
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --describe --topic clario360.events.<TOPIC_NAME>
```

Create all standard cross-suite topics if multiple are missing:

```bash
for TOPIC in cyber workflow notification audit iam data acta lex visus cross-suite; do
  echo "Creating clario360.events.$TOPIC..."
  kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
    kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
    --create \
    --topic clario360.events.$TOPIC \
    --partitions 6 \
    --replication-factor 3 \
    --config retention.ms=604800000 \
    --config cleanup.policy=delete \
    --config max.message.bytes=1048576 \
    --if-not-exists
done
```

### Resolution B: Fix Kafka ACLs

If a service cannot produce to or consume from a topic due to ACL restrictions:

```bash
# Grant produce permission to a service
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --add \
  --allow-principal User:clario360-<PRODUCER_SERVICE> \
  --operation Write \
  --topic clario360.events.<TOPIC_NAME>

# Grant consume permission to a service
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --add \
  --allow-principal User:clario360-<CONSUMER_SERVICE> \
  --operation Read \
  --topic clario360.events.<TOPIC_NAME>

# Grant consumer group permission
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --add \
  --allow-principal User:clario360-<CONSUMER_SERVICE> \
  --operation Read \
  --group clario360-<CONSUMER_SERVICE>-consumer

# Verify updated ACLs
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-acls.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --list --topic clario360.events.<TOPIC_NAME>
```

### Resolution C: Reset Consumer Group Offset

If a consumer is stuck or needs to reprocess events:

```bash
# Reset to latest (skip all unprocessed messages)
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --group clario360-<SERVICE>-consumer \
  --topic clario360.events.<TOPIC_NAME> \
  --reset-offsets --to-latest --execute

# OR reset to a specific timestamp (reprocess from that point)
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --group clario360-<SERVICE>-consumer \
  --topic clario360.events.<TOPIC_NAME> \
  --reset-offsets --to-datetime "2026-03-08T00:00:00.000" --execute

# OR reset to the beginning (reprocess all events)
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --group clario360-<SERVICE>-consumer \
  --topic clario360.events.<TOPIC_NAME> \
  --reset-offsets --to-earliest --execute
```

**Important**: The consumer service must be stopped before resetting offsets. Scale to 0 replicas first:

```bash
kubectl scale deployment/<CONSUMER_SERVICE> -n clario360 --replicas=0
# ... reset offsets ...
kubectl scale deployment/<CONSUMER_SERVICE> -n clario360 --replicas=3
kubectl rollout status deployment/<CONSUMER_SERVICE> -n clario360 --timeout=120s
```

### Resolution D: Fix Event Schema Incompatibility

If consumer services cannot deserialize events due to schema changes:

**1. Register the updated schema:**

```bash
kubectl port-forward -n kafka svc/schema-registry 8081:8081 &
PF_PID=$!
sleep 2

# Register a new schema version (example for JSON schema)
curl -s -X POST http://localhost:8081/subjects/clario360.events.<TOPIC_NAME>-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"object\",\"properties\":{\"event_type\":{\"type\":\"string\"},\"source\":{\"type\":\"string\"},\"timestamp\":{\"type\":\"string\"},\"data\":{\"type\":\"object\"}}}"
  }' | jq .

kill $PF_PID
```

**2. If schema compatibility mode is blocking registration, adjust it temporarily:**

```bash
kubectl port-forward -n kafka svc/schema-registry 8081:8081 &
PF_PID=$!
sleep 2

# Check current compatibility level
curl -s http://localhost:8081/config/clario360.events.<TOPIC_NAME>-value | jq .

# Set to NONE temporarily (use with caution)
curl -s -X PUT http://localhost:8081/config/clario360.events.<TOPIC_NAME>-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "NONE"}' | jq .

# Register the schema, then restore compatibility
curl -s -X PUT http://localhost:8081/config/clario360.events.<TOPIC_NAME>-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "BACKWARD"}' | jq .

kill $PF_PID
```

**3. Restart consumer services to pick up the new schema:**

```bash
kubectl rollout restart deployment/<CONSUMER_SERVICE> -n clario360
kubectl rollout status deployment/<CONSUMER_SERVICE> -n clario360 --timeout=120s
```

### Resolution E: Restart Consumer Services

If consumer groups show no active members or are in a bad state:

```bash
# Restart a specific consumer service
kubectl rollout restart deployment/<CONSUMER_SERVICE> -n clario360
kubectl rollout status deployment/<CONSUMER_SERVICE> -n clario360 --timeout=120s

# Restart all services that consume cross-suite events
for SERVICE in audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout restart deployment/$SERVICE -n clario360
done

# Wait for all rollouts to complete
for SERVICE in audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl rollout status deployment/$SERVICE -n clario360 --timeout=120s
done
```

### Resolution F: Fix Kafka Broker Configuration

If the issue is at the broker level:

```bash
# Check broker configuration for relevant settings
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-configs.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --entity-type brokers --entity-default --describe | grep -E "auto.create|message.max|retention"

# Enable auto topic creation if disabled (temporary fix)
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-configs.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --entity-type brokers --entity-default \
  --alter --add-config auto.create.topics.enable=true

# Increase max message size if events are being rejected
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-configs.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
  --entity-type topics --entity-name clario360.events.<TOPIC_NAME> \
  --alter --add-config max.message.bytes=5242880
```

### Resolution G: Fix Network Connectivity Between Services and Kafka

```bash
# Test connectivity from a service pod to Kafka
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') -- \
  nc -zv kafka.kafka.svc.cluster.local 9092

# Check if a NetworkPolicy is blocking traffic
kubectl get networkpolicy -n clario360
kubectl get networkpolicy -n kafka

# If a NetworkPolicy is blocking, check its rules
kubectl describe networkpolicy -n kafka
kubectl describe networkpolicy -n clario360

# Test DNS resolution
kubectl exec -n clario360 -it $(kubectl get pod -n clario360 -l app=<SERVICE> -o jsonpath='{.items[0].metadata.name}') -- \
  nslookup kafka.kafka.svc.cluster.local
```

---

## Verification

After applying a resolution, verify cross-suite events are flowing:

```bash
# 1. Verify all required topics exist
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 --list | grep clario360

# 2. Check consumer group lag is decreasing
for SERVICE in audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  echo "=== $SERVICE ==="
  kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
    kafka-consumer-groups.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 \
    --describe --group clario360-$SERVICE-consumer 2>&1 | tail -5
  echo ""
done

# 3. Produce a test event and verify it is consumed
kubectl exec -n kafka -it $(kubectl get pod -n kafka -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  bash -c 'echo "{\"event_type\":\"test.verification\",\"source\":\"ts-010-runbook\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"data\":{\"message\":\"post-fix verification\"}}" | kafka-console-producer.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 --topic clario360.events.cross-suite'

# Wait a few seconds for consumption
sleep 5

# Check consumer service logs for the test event
kubectl logs -n clario360 -l app=notification-service --since=1m --timestamps | grep -i "test.verification"

# 4. Verify no producer errors in service logs
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  ERRORS=$(kubectl logs -n clario360 -l app=$SERVICE --since=5m --timestamps 2>/dev/null | grep -i -c -E "kafka.*error|produce.*fail|publish.*fail")
  echo "$SERVICE: $ERRORS producer errors in last 5m"
done

# 5. Verify no consumer errors in service logs
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  ERRORS=$(kubectl logs -n clario360 -l app=$SERVICE --since=5m --timestamps 2>/dev/null | grep -i -c -E "consume.*error|deserializ.*error|offset.*error")
  echo "$SERVICE: $ERRORS consumer errors in last 5m"
done

# 6. Verify service health
for SERVICE in api-gateway iam-service audit-service workflow-engine notification-service cyber-service data-service acta-service lex-service visus-service; do
  kubectl port-forward -n clario360 svc/$SERVICE 8080:8080 &
  PF_PID=$!
  sleep 2
  HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz)
  READY=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/readyz)
  echo "$SERVICE: healthz=$HEALTH readyz=$READY"
  kill $PF_PID 2>/dev/null
done

# 7. Check Kafka broker health
kubectl get pods -n kafka -l app=kafka -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,RESTARTS:.status.containerStatuses[0].restartCount
```

---

## Post-Incident Checklist

- [ ] Confirm all cross-suite event topics exist with correct partition count and replication factor.
- [ ] Confirm all consumer groups have active members and lag is at zero or decreasing.
- [ ] Confirm no producer or consumer errors in any service logs.
- [ ] Confirm all services pass `/healthz` and `/readyz` checks.
- [ ] Verify Prometheus Kafka consumer lag alerts have cleared in Alertmanager.
- [ ] If topics were created, document them and add to the infrastructure-as-code (Terraform/Helm).
- [ ] If ACLs were modified, document changes and update ACL management configuration.
- [ ] If consumer offsets were reset, verify no duplicate processing occurred or handle idempotently.
- [ ] If schema was updated, ensure backward/forward compatibility is maintained.
- [ ] Test the specific cross-suite event flow that was broken with an end-to-end test.
- [ ] Document root cause and corrective actions.
- [ ] Review Kafka monitoring and alerting for gaps that allowed this issue to go undetected.
- [ ] Consider adding end-to-end event flow health checks (synthetic events).

---

## Related Links

| Resource                         | Link                                                                     |
|----------------------------------|--------------------------------------------------------------------------|
| Kafka Operations Documentation   | https://kafka.apache.org/documentation/#operations                       |
| Confluent Schema Registry Docs   | https://docs.confluent.io/platform/current/schema-registry/             |
| Grafana Dashboards               | https://grafana.clario360.internal/dashboards                            |
| Alertmanager                     | https://alertmanager.clario360.internal                                  |
| IR-001 Service Outage            | [../incident-response/IR-001-service-outage.md](../incident-response/IR-001-service-outage.md) |
| IR-003 Kafka Failure             | [../incident-response/IR-003-kafka-failure.md](../incident-response/IR-003-kafka-failure.md) |
| TS-009 Audit Chain Broken        | [TS-009-audit-chain-broken.md](./TS-009-audit-chain-broken.md)           |
