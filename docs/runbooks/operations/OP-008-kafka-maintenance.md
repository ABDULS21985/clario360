# OP-008: Kafka Topic & Consumer Maintenance

| Field | Value |
|-------|-------|
| **Runbook ID** | OP-008 |
| **Title** | Kafka Topic & Consumer Maintenance |
| **Frequency** | Weekly |
| **Owner** | Platform Infrastructure Team |
| **Last Updated** | 2026-03-08 |
| **Estimated Duration** | 1–2 hours |
| **Risk Level** | Low (monitoring) to Medium (retention/partition changes) |
| **Approval Required** | No for monitoring; Yes for topic deletion or retention changes |
| **Maintenance Window** | Monitoring: anytime; Changes: off-peak hours |

## Summary

This runbook covers recurring Kafka maintenance tasks for the Clario 360 platform:

1. Check broker health and disk usage
2. Review consumer group lag across all groups
3. Clean up unused/empty topics
4. Adjust retention settings for high-volume topics
5. Verify replication factor on critical topics
6. Monitor partition distribution across brokers

### Kafka Cluster Details

| Setting | Value |
|---------|-------|
| Namespace | `kafka` |
| Brokers | `kafka-0`, `kafka-1`, `kafka-2` |
| Bootstrap servers | `kafka-0.kafka-headless.kafka.svc.cluster.local:9092` |
| ZooKeeper (if applicable) | `zookeeper.kafka.svc.cluster.local:2181` |
| Replication factor (default) | 3 |
| Min in-sync replicas | 2 |

### Known Clario 360 Topics

| Topic | Producers | Consumers | Volume |
|-------|-----------|-----------|--------|
| `audit.events` | All services | audit-service | High |
| `workflow.tasks` | workflow-engine | workflow-engine, notification-service | Medium |
| `workflow.events` | workflow-engine | notification-service, audit-service | Medium |
| `notifications` | notification-service, workflow-engine | notification-service | Medium |
| `cyber.findings` | cyber-service | notification-service, data-service | Medium |
| `cyber.scans` | cyber-service | cyber-service | Low |
| `data.lineage` | data-service | visus-service | Medium |
| `data.quality` | data-service | notification-service | Low |
| `acta.documents` | acta-service | notification-service, lex-service | Low |
| `lex.obligations` | lex-service | notification-service, acta-service | Low |
| `visus.reports` | visus-service | notification-service | Low |
| `iam.events` | iam-service | audit-service | Medium |
| `platform.events` | All services | Multiple | High |
| `dead-letter` | All services | ops monitoring | Low |

## Prerequisites

```bash
export KAFKA_NS=kafka
export BOOTSTRAP=kafka-0.kafka-headless.kafka.svc.cluster.local:9092
export NAMESPACE=clario360
```

Verify Kafka CLI tools are available:

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh --version
```

---

## Step 1: Check Broker Health and Disk Usage

### 1a. Verify all brokers are running

```bash
kubectl -n $KAFKA_NS get pods -l app=kafka -o wide
```

All three broker pods should show `Running` and `1/1` ready.

### 1b. Check broker health via metadata request

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --broker-info 2>/dev/null || \
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-broker-api-versions.sh \
  --bootstrap-server $BOOTSTRAP 2>&1 | grep -E "^[a-z].*\(id:" | wc -l
```

Expected: 3 brokers responding.

### 1c. Check disk usage on each broker

```bash
for broker in kafka-0 kafka-1 kafka-2; do
  echo "=== $broker ==="
  kubectl -n $KAFKA_NS exec $broker -- df -h /var/lib/kafka/data
  echo ""
done
```

**Alert threshold:** If any broker exceeds 75% disk usage, take action:
- Reduce retention on high-volume topics (see Step 4)
- Delete old/empty topics (see Step 3)
- Scale broker disk (see SC-003)

### 1d. Check log segment sizes per topic

```bash
for broker in kafka-0 kafka-1 kafka-2; do
  echo "=== $broker: top 10 topics by disk usage ==="
  kubectl -n $KAFKA_NS exec $broker -- du -sh /var/lib/kafka/data/*/ 2>/dev/null | sort -rh | head -10
  echo ""
done
```

### 1e. Check controller status

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --controller 2>/dev/null || \
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe --topic __consumer_offsets | head -1
```

---

## Step 2: Review Consumer Group Lag

### 2a. List all consumer groups

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --list
```

### 2b. Check lag for each consumer group

```bash
GROUPS=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --list 2>/dev/null)

for group in $GROUPS; do
  echo "=== Consumer Group: $group ==="
  kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe --group $group 2>/dev/null | head -20
  echo ""
done
```

### 2c. Identify groups with significant lag

```bash
echo "=== Consumer groups with lag > 1000 ==="
for group in $GROUPS; do
  LAG=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe --group $group 2>/dev/null | \
    awk 'NR>1 {sum += $6} END {print sum+0}')
  if [ "$LAG" -gt 1000 ] 2>/dev/null; then
    echo "$group: total lag = $LAG"
  fi
done
```

**Action if lag is growing:**
1. Check if the consumer service is running: `kubectl -n $NAMESPACE get pods -l app=<service>`
2. Check consumer service logs: `kubectl -n $NAMESPACE logs deployment/<service> --tail=50`
3. If lag is due to slow processing, consider scaling the consumer (see SC-001)

### 2d. Check for inactive consumer groups

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --list --state | grep -i "empty\|dead"
```

---

## Step 3: Clean Up Unused/Empty Topics

### 3a. List all topics

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --list
```

### 3b. Identify topics with zero messages

```bash
ALL_TOPICS=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --list 2>/dev/null | grep -v "^__")

for topic in $ALL_TOPICS; do
  OFFSET_SUM=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-run-class.sh kafka.tools.GetOffsetShell \
    --broker-list $BOOTSTRAP \
    --topic $topic 2>/dev/null | awk -F: '{sum += $3} END {print sum+0}')
  if [ "$OFFSET_SUM" -eq 0 ] 2>/dev/null; then
    echo "EMPTY: $topic (0 messages)"
  fi
done
```

### 3c. Identify topics with no active consumers

```bash
for topic in $ALL_TOPICS; do
  CONSUMERS=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
    --bootstrap-server $BOOTSTRAP \
    --list 2>/dev/null | while read group; do
      kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
        --bootstrap-server $BOOTSTRAP \
        --describe --group $group 2>/dev/null | grep -q "$topic" && echo "$group"
    done)
  if [ -z "$CONSUMERS" ]; then
    echo "NO CONSUMERS: $topic"
  fi
done
```

### 3d. Delete unused topics

> **Warning:** Only delete topics after confirming they are not used by any service. Cross-reference with the known topics table above.

```bash
# Delete a specific unused topic
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --delete --topic <topic-name>
```

### 3e. Delete inactive consumer groups

```bash
# Delete a specific inactive consumer group
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --delete --group <group-name>
```

---

## Step 4: Adjust Retention Settings for High-Volume Topics

### 4a. Check current retention settings

```bash
CRITICAL_TOPICS="audit.events workflow.tasks workflow.events notifications cyber.findings data.lineage platform.events"

for topic in $CRITICAL_TOPICS; do
  echo "=== $topic ==="
  kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
    --bootstrap-server $BOOTSTRAP \
    --entity-type topics \
    --entity-name $topic \
    --describe | grep -E "retention|segment|cleanup"
  echo ""
done
```

### 4b. Recommended retention settings

| Topic | retention.ms | retention.bytes | segment.bytes | cleanup.policy |
|-------|-------------|-----------------|---------------|----------------|
| `audit.events` | 2592000000 (30d) | -1 (unlimited) | 1073741824 (1GB) | delete |
| `workflow.tasks` | 604800000 (7d) | -1 | 536870912 (512MB) | delete |
| `workflow.events` | 604800000 (7d) | -1 | 536870912 (512MB) | delete |
| `notifications` | 259200000 (3d) | -1 | 268435456 (256MB) | delete |
| `cyber.findings` | 2592000000 (30d) | -1 | 1073741824 (1GB) | delete |
| `data.lineage` | 2592000000 (30d) | -1 | 536870912 (512MB) | compact |
| `platform.events` | 604800000 (7d) | -1 | 536870912 (512MB) | delete |
| `dead-letter` | 7776000000 (90d) | -1 | 268435456 (256MB) | delete |

### 4c. Update retention for a topic

```bash
# Example: set audit.events to 30-day retention
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server $BOOTSTRAP \
  --entity-type topics \
  --entity-name audit.events \
  --alter \
  --add-config "retention.ms=2592000000,segment.bytes=1073741824,cleanup.policy=delete"

# Example: set notifications to 3-day retention
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server $BOOTSTRAP \
  --entity-type topics \
  --entity-name notifications \
  --alter \
  --add-config "retention.ms=259200000,segment.bytes=268435456,cleanup.policy=delete"

# Example: set data.lineage to compacted
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server $BOOTSTRAP \
  --entity-type topics \
  --entity-name data.lineage \
  --alter \
  --add-config "retention.ms=2592000000,segment.bytes=536870912,cleanup.policy=compact"
```

### 4d. Trigger log segment cleanup (if disk is critical)

```bash
# Force log cleanup on a specific topic by temporarily setting retention very low, then restoring
# WARNING: This will delete data — only use in emergencies
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server $BOOTSTRAP \
  --entity-type topics \
  --entity-name <topic-name> \
  --alter \
  --add-config "retention.ms=1000"

# Wait for cleaner to run (check logs)
sleep 30

# Restore original retention
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-configs.sh \
  --bootstrap-server $BOOTSTRAP \
  --entity-type topics \
  --entity-name <topic-name> \
  --alter \
  --add-config "retention.ms=<original-value>"
```

---

## Step 5: Verify Replication Factor on Critical Topics

### 5a. Check replication factor and ISR for all topics

```bash
CRITICAL_TOPICS="audit.events workflow.tasks workflow.events notifications cyber.findings cyber.scans data.lineage data.quality acta.documents lex.obligations visus.reports iam.events platform.events dead-letter"

for topic in $CRITICAL_TOPICS; do
  echo "=== $topic ==="
  kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe --topic $topic 2>/dev/null | head -5
  echo ""
done
```

### 5b. Identify under-replicated partitions

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe --under-replicated-partitions
```

If any partitions are under-replicated:

1. Check if a broker is down: `kubectl -n $KAFKA_NS get pods -l app=kafka`
2. Check broker logs: `kubectl -n $KAFKA_NS logs kafka-<id> --tail=50`
3. If a broker is restarting, wait for it to rejoin and catch up
4. Refer to [IR-003: Kafka Failure](../incident-response/IR-003-kafka-failure.md) if brokers are persistently down

### 5c. Identify topics with replication factor < 3

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe 2>/dev/null | grep "ReplicationFactor:" | grep -v "ReplicationFactor: 3"
```

### 5d. Increase replication factor (if needed)

```bash
# Create a reassignment JSON for the topic
# Example: increase "notifications" from RF=2 to RF=3
cat > /tmp/reassignment.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "notifications", "partition": 0, "replicas": [0, 1, 2]},
    {"topic": "notifications", "partition": 1, "replicas": [1, 2, 0]},
    {"topic": "notifications", "partition": 2, "replicas": [2, 0, 1]}
  ]
}
EOF

kubectl cp /tmp/reassignment.json $KAFKA_NS/kafka-0:/tmp/reassignment.json

kubectl -n $KAFKA_NS exec kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server $BOOTSTRAP \
  --reassignment-json-file /tmp/reassignment.json \
  --execute

# Verify reassignment progress
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server $BOOTSTRAP \
  --reassignment-json-file /tmp/reassignment.json \
  --verify
```

---

## Step 6: Monitor Partition Distribution Across Brokers

### 6a. Check partition count per broker

```bash
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe 2>/dev/null | \
  grep -oP 'Leader: \K[0-9]+' | sort | uniq -c | sort -rn
```

This shows how many partitions each broker leads. The distribution should be roughly even.

### 6b. Check partition leader skew

```bash
echo "=== Partition distribution (Leader / Replicas) ==="
for broker_id in 0 1 2; do
  LEADER_COUNT=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe 2>/dev/null | grep "Leader: $broker_id" | wc -l)
  REPLICA_COUNT=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe 2>/dev/null | grep -oP "Replicas: [^\t]+" | grep -c "$broker_id")
  echo "Broker $broker_id: $LEADER_COUNT leaders, $REPLICA_COUNT replicas"
done
```

### 6c. Trigger preferred leader election (if skewed)

```bash
# This re-elects the preferred leader for all partitions
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-leader-election.sh \
  --bootstrap-server $BOOTSTRAP \
  --election-type preferred \
  --all-topic-partitions
```

### 6d. Check overall cluster metrics

```bash
echo "=== Topic count ==="
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --list 2>/dev/null | grep -v "^__" | wc -l

echo "=== Total partition count ==="
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe 2>/dev/null | grep "Partition:" | wc -l

echo "=== Consumer group count ==="
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --list 2>/dev/null | wc -l
```

---

## Verification

After completing maintenance:

```bash
# 1. All brokers healthy
kubectl -n $KAFKA_NS get pods -l app=kafka
for broker in kafka-0 kafka-1 kafka-2; do
  kubectl -n $KAFKA_NS exec $broker -- kafka-broker-api-versions.sh \
    --bootstrap-server $BOOTSTRAP 2>&1 | head -1
done

# 2. No under-replicated partitions
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-topics.sh \
  --bootstrap-server $BOOTSTRAP \
  --describe --under-replicated-partitions

# 3. All consumer groups active and lag acceptable
kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server $BOOTSTRAP \
  --list 2>/dev/null | while read group; do
  TOTAL_LAG=$(kubectl -n $KAFKA_NS exec kafka-0 -- kafka-consumer-groups.sh \
    --bootstrap-server $BOOTSTRAP \
    --describe --group $group 2>/dev/null | awk 'NR>1 {sum+=$6} END {print sum+0}')
  echo "$group: lag=$TOTAL_LAG"
done

# 4. Disk usage acceptable (<75%)
for broker in kafka-0 kafka-1 kafka-2; do
  echo "=== $broker ==="
  kubectl -n $KAFKA_NS exec $broker -- df -h /var/lib/kafka/data | tail -1
done

# 5. Check Grafana Kafka dashboard
echo "Review: https://grafana.clario360.io/d/kafka-overview"
```

---

## Related Links

- [IR-003: Kafka Failure](../incident-response/IR-003-kafka-failure.md)
- [SC-003: Kafka Scaling](../scaling/SC-003-kafka-scaling.md)
- [TS-003: Missing Events](../troubleshooting/TS-003-missing-events.md)
- [TS-010: Cross-Suite Events Not Flowing](../troubleshooting/TS-010-cross-suite-events-not-flowing.md)
- [Grafana — Kafka Overview](https://grafana.clario360.io/d/kafka-overview)
