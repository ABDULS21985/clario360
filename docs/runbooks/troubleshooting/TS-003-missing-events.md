# TS-003: Kafka Event Loss Investigation

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | TS-003                                     |
| **Title**        | Kafka Event Loss Investigation             |
| **Severity**     | P1 - Critical                              |
| **Services**     | All services (producers/consumers), Kafka  |
| **Last Updated** | 2026-03-08                                 |
| **Author**       | Platform Engineering                       |
| **Review Cycle** | Quarterly                                  |

---

## Summary

This runbook covers the investigation and resolution of missing or lost Kafka events in the Clario 360 platform. Events flow between services via Kafka topics for audit logging, notifications, workflow orchestration, and data pipeline coordination. Event loss can result from producer failures, topic misconfiguration, consumer group offset issues, deserialization errors, or dead letter queue accumulation. Follow the diagnosis steps to identify where events are being dropped.

---

## Symptoms

- Audit logs are missing entries for actions that were performed.
- Notifications are not being delivered for events that should trigger them.
- Workflow steps are not progressing despite upstream events completing.
- Data pipelines are not triggered by expected events.
- Consumer group lag is zero but expected events are missing from the target system.
- Dead letter queue topic contains unexpected volumes of messages.
- Monitoring alerts fire for event processing SLA breaches.

---

## Diagnosis Steps

### Step 1: Verify Producer Is Sending Events

```bash
# Check producer-side service logs for send errors
kubectl -n clario360 logs deploy/api-gateway --tail=300 | grep -i -E 'kafka|produce|send.*event|publish'
kubectl -n clario360 logs deploy/iam-service --tail=300 | grep -i -E 'kafka|produce|send.*event|publish'
kubectl -n clario360 logs deploy/workflow-engine --tail=300 | grep -i -E 'kafka|produce|send.*event|publish'
kubectl -n clario360 logs deploy/data-service --tail=300 | grep -i -E 'kafka|produce|send.*event|publish'
kubectl -n clario360 logs deploy/cyber-service --tail=300 | grep -i -E 'kafka|produce|send.*event|publish'
```

```bash
# Check producer metrics (if the service exposes Kafka producer metrics)
kubectl -n clario360 port-forward svc/api-gateway 8080:8080
curl -s http://localhost:8080/metrics | grep -E 'kafka_producer|messages_sent|produce_errors'
```

```bash
# Consume from the topic in real-time to verify messages are arriving
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events \
  --from-beginning \
  --max-messages 10 \
  --property print.timestamp=true \
  --property print.key=true \
  --property print.headers=true

# Count total messages in the topic
kubectl -n kafka exec -it kafka-0 -- kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events \
  --time -1

# Count messages from the beginning (earliest offset)
kubectl -n kafka exec -it kafka-0 -- kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events \
  --time -2
```

The difference between the `-1` (latest) and `-2` (earliest) offsets across all partitions gives the total available messages.

### Step 2: Check Topic Existence and Partition Count

```bash
# List all topics
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list

# Describe the specific topic
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --topic clario360.events

# Check all Clario 360 topics
for topic in clario360.events clario360.audit.events clario360.notification.events clario360.workflow.events clario360.pipeline.events; do
  echo "=== $topic ==="
  kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
    --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
    --describe --topic $topic
done
```

Verify:
- The topic exists (if it was auto-deleted or never created, events are silently dropped by some producer configurations).
- The partition count matches or exceeds the consumer instance count (otherwise some consumers will be idle).
- The replication factor is at least 2 for durability.
- The retention period (`retention.ms`) has not expired the messages you are looking for.

### Step 3: Check Consumer Group Offset vs Latest Offset

```bash
# List all consumer groups
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list

# Describe each relevant consumer group
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-audit-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-notification-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-workflow-engine

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-data-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-cyber-service
```

Key columns in the output:
- **CURRENT-OFFSET**: where the consumer has read up to.
- **LOG-END-OFFSET**: the latest message offset in the partition.
- **LAG**: the difference (LOG-END-OFFSET - CURRENT-OFFSET). Non-zero lag means unprocessed messages.
- **CONSUMER-ID**: if empty, the partition has no active consumer assigned.

If CURRENT-OFFSET equals LOG-END-OFFSET (lag = 0) but events are still missing, the issue is downstream (processing failure, not delivery failure).

### Step 4: Check for Deserialization Errors in Consumer Logs

```bash
# Check audit-service consumer logs
kubectl -n clario360 logs deploy/audit-service --tail=500 | grep -i -E 'deserializ|unmarshal|json.*error|avro|schema|malformed|invalid.*message'

# Check notification-service consumer logs
kubectl -n clario360 logs deploy/notification-service --tail=500 | grep -i -E 'deserializ|unmarshal|json.*error|avro|schema|malformed|invalid.*message'

# Check workflow-engine consumer logs
kubectl -n clario360 logs deploy/workflow-engine --tail=500 | grep -i -E 'deserializ|unmarshal|json.*error|avro|schema|malformed|invalid.*message'

# Check data-service consumer logs
kubectl -n clario360 logs deploy/data-service --tail=500 | grep -i -E 'deserializ|unmarshal|json.*error|avro|schema|malformed|invalid.*message'

# Check all services at once for broader search
for svc in audit-service notification-service workflow-engine data-service cyber-service acta-service lex-service visus-service; do
  echo "=== $svc ==="
  kubectl -n clario360 logs deploy/$svc --tail=200 | grep -i -E 'deserializ|unmarshal|invalid.*event|unknown.*type' | tail -5
done
```

### Step 5: Check Dead Letter Queue (DLQ)

```bash
# List DLQ topics
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list | grep -i dlq

# Check message count in DLQ topics
for dlq_topic in clario360.events.dlq clario360.audit.events.dlq clario360.notification.events.dlq clario360.workflow.events.dlq clario360.pipeline.events.dlq; do
  echo "=== $dlq_topic ==="
  kubectl -n kafka exec -it kafka-0 -- kafka-run-class.sh kafka.tools.GetOffsetShell \
    --broker-list kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
    --topic $dlq_topic \
    --time -1 2>/dev/null || echo "Topic does not exist"
done

# Read messages from the DLQ to understand failure reasons
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events.dlq \
  --from-beginning \
  --max-messages 10 \
  --property print.timestamp=true \
  --property print.key=true \
  --property print.headers=true
```

### Step 6: Check Kafka Broker Health

```bash
# Verify Kafka broker pods are running
kubectl -n kafka get pods

# Check Kafka broker logs for errors
kubectl -n kafka logs kafka-0 --tail=200 | grep -i -E 'error|warn|exception|oom|disk'
kubectl -n kafka logs kafka-1 --tail=200 | grep -i -E 'error|warn|exception|oom|disk'
kubectl -n kafka logs kafka-2 --tail=200 | grep -i -E 'error|warn|exception|oom|disk'

# Check under-replicated partitions
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions

# Check broker disk usage
kubectl -n kafka exec -it kafka-0 -- df -h /var/lib/kafka/data
kubectl -n kafka exec -it kafka-1 -- df -h /var/lib/kafka/data
kubectl -n kafka exec -it kafka-2 -- df -h /var/lib/kafka/data
```

### Step 7: Check Topic Retention Configuration

```bash
# Check retention settings on the topic
kubectl -n kafka exec -it kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --entity-type topics \
  --entity-name clario360.events \
  --describe

# If retention.ms is too low, messages may have been deleted before consumers read them
# Default retention is 7 days (604800000 ms)
```

---

## Resolution Steps

### Resolution: Reset Consumer Group Offset

Use this when the consumer group offset has skipped past messages (e.g., due to a rebalance or crash).

```bash
# IMPORTANT: Stop the consumer service first
kubectl -n clario360 scale deploy/<consumer-service> --replicas=0

# Reset to the earliest available offset (replay all retained messages)
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --group clario360-<service-name> \
  --topic clario360.events \
  --reset-offsets \
  --to-earliest \
  --execute

# Or reset to a specific timestamp (replay from a point in time)
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --group clario360-<service-name> \
  --topic clario360.events \
  --reset-offsets \
  --to-datetime "2026-03-07T00:00:00.000" \
  --execute

# Or reset to a specific offset on a partition
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --group clario360-<service-name> \
  --topic clario360.events:0 \
  --reset-offsets \
  --to-offset 12345 \
  --execute

# Restart the consumer service
kubectl -n clario360 scale deploy/<consumer-service> --replicas=2
kubectl -n clario360 rollout status deploy/<consumer-service>
```

### Resolution: Replay Events from DLQ

```bash
# Replay messages from DLQ back to the main topic
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events.dlq \
  --from-beginning \
  --max-messages 1000 \
  --property print.key=true \
  --property key.separator="|" | \
kubectl -n kafka exec -i kafka-0 -- kafka-console-producer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events \
  --property parse.key=true \
  --property key.separator="|"

# After replay, purge the DLQ (by setting retention to 1ms temporarily)
kubectl -n kafka exec -it kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --entity-type topics \
  --entity-name clario360.events.dlq \
  --alter --add-config retention.ms=1000

# Wait a moment for cleanup, then restore retention
sleep 10
kubectl -n kafka exec -it kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --entity-type topics \
  --entity-name clario360.events.dlq \
  --alter --add-config retention.ms=604800000
```

### Resolution: Fix Serialization / Deserialization

1. **Identify the problematic message format** by examining DLQ messages:

```bash
kubectl -n kafka exec -it kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events.dlq \
  --from-beginning \
  --max-messages 5 \
  --property print.headers=true
```

2. **If the producer is sending an incompatible format**, update the producer service configuration:

```bash
# Check the producer service's event serialization config
kubectl -n clario360 get configmap <service>-config -o yaml

# Update the serialization format if needed
kubectl -n clario360 patch configmap <service>-config --type='merge' -p='{"data":{"EVENT_SERIALIZATION_FORMAT":"json"}}'

# Restart the producer service
kubectl -n clario360 rollout restart deploy/<producer-service>
kubectl -n clario360 rollout status deploy/<producer-service>
```

3. **If the consumer cannot handle a new event type**, update and redeploy the consumer:

```bash
# Check the consumer service configuration
kubectl -n clario360 get configmap <consumer-service>-config -o yaml

# After updating the consumer code to handle the new event type, redeploy
kubectl -n clario360 rollout restart deploy/<consumer-service>
kubectl -n clario360 rollout status deploy/<consumer-service>
```

### Resolution: Create Missing Topic

```bash
# Create the missing topic with appropriate settings
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --create \
  --topic clario360.events \
  --partitions 6 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config cleanup.policy=delete \
  --config min.insync.replicas=2
```

### Resolution: Increase Topic Retention

```bash
# Increase retention to 14 days (if messages are being deleted too quickly)
kubectl -n kafka exec -it kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --entity-type topics \
  --entity-name clario360.events \
  --alter --add-config retention.ms=1209600000
```

### Resolution: Fix Under-Replicated Partitions

```bash
# Check which brokers are behind
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions

# Restart the lagging broker
kubectl -n kafka delete pod kafka-<broker-id>

# Verify replication catches up
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions
```

---

## Verification

After applying the resolution, verify the fix:

```bash
# 1. Verify consumer groups show zero or decreasing lag
kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-audit-service

kubectl -n kafka exec -it kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --group clario360-notification-service

# 2. Produce a test event and confirm it is consumed
kubectl -n kafka exec -it kafka-0 -- bash -c 'echo "{\"type\":\"test\",\"timestamp\":\"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'\"}" | kafka-console-producer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events'

# 3. Check consumer service logs for the test event
kubectl -n clario360 logs deploy/audit-service --tail=20 | grep -i test
kubectl -n clario360 logs deploy/notification-service --tail=20 | grep -i test

# 4. Verify DLQ is not growing
kubectl -n kafka exec -it kafka-0 -- kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events.dlq \
  --time -1

# 5. Verify no deserialization errors in consumer logs
for svc in audit-service notification-service workflow-engine data-service; do
  echo "=== $svc ==="
  kubectl -n clario360 logs deploy/$svc --tail=100 --since=5m | grep -i -E 'deserializ|unmarshal|error' | wc -l
done

# 6. Verify all consumer services are healthy
for svc in audit-service notification-service workflow-engine data-service cyber-service; do
  echo "--- $svc ---"
  kubectl -n clario360 exec deploy/$svc -- curl -s http://localhost:8080/healthz
done

# 7. Check that no partitions are under-replicated
kubectl -n kafka exec -it kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe --under-replicated-partitions
```

---

## Related Links

- [TS-001: API Latency Investigation](./TS-001-slow-api-responses.md)
- [TS-002: Data Pipeline Failure Investigation](./TS-002-failed-pipelines.md)
- [TS-004: Authentication Issue Debugging](./TS-004-auth-failures.md)
- [TS-005: WebSocket Connectivity Issues](./TS-005-websocket-disconnects.md)
- Grafana Kafka Dashboard: `/d/kafka-overview/kafka-cluster-overview`
- Grafana Consumer Lag Dashboard: `/d/kafka-lag/kafka-consumer-lag`
- Kafka Documentation: https://kafka.apache.org/documentation/
