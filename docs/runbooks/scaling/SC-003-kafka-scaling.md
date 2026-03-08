# SC-003: Kafka Scaling

| Field            | Value                                      |
|------------------|--------------------------------------------|
| **Runbook ID**   | SC-003                                     |
| **Title**        | Kafka Scaling                              |
| **Category**     | Scaling                                    |
| **Severity**     | High                                       |
| **Author**       | Platform Engineering                       |
| **Created**      | 2026-03-08                                 |
| **Last Updated** | 2026-03-08                                 |
| **Review Cycle** | Quarterly                                  |
| **Platform**     | GCP (GKE)                                  |
| **Namespace**    | kafka                                      |

---

## Summary

This runbook covers scaling Apache Kafka for the Clario 360 platform. It includes adding new brokers to the cluster, rebalancing partitions, increasing partition counts for high-throughput topics, adjusting replication factors, and monitoring broker performance after scaling operations. The Kafka cluster runs in the `kafka` namespace on GKE.

### Platform Topics

| Topic                               | Partitions | Replication | Producers                              | Consumers                               |
|--------------------------------------|-----------|-------------|----------------------------------------|------------------------------------------|
| clario360.audit.events              | 12        | 3           | All services                           | audit-service                            |
| clario360.notifications             | 6         | 3           | workflow-engine, cyber-service         | notification-service                     |
| clario360.workflow.commands         | 6         | 3           | api-gateway, iam-service               | workflow-engine                          |
| clario360.workflow.events           | 12        | 3           | workflow-engine                        | notification-service, audit-service      |
| clario360.cyber.findings            | 12        | 3           | cyber-service                          | notification-service, data-service       |
| clario360.data.ingestion            | 12        | 3           | api-gateway, data-service              | data-service                             |
| clario360.lex.regulatory-updates    | 6         | 3           | lex-service                            | notification-service, acta-service       |
| clario360.acta.document-events      | 6         | 3           | acta-service                           | audit-service, notification-service      |
| clario360.visus.report-requests     | 6         | 3           | api-gateway, visus-service             | visus-service                            |
| clario360.events                    | 12        | 3           | All services                           | Multiple consumers                       |

---

## Prerequisites

- `kubectl` CLI configured with cluster credentials
- Access to the `kafka` namespace
- Kafka CLI tools available (`kafka-topics.sh`, `kafka-reassign-partitions.sh`, etc.)
- Understanding of current Kafka cluster topology
- Sufficient node capacity in the GKE cluster for new brokers (see SC-004)

### Verify Cluster Access

```bash
# Verify Kafka pods are running
kubectl get pods -n kafka -l app=kafka

# Get the list of Kafka brokers
kubectl get pods -n kafka -l app=kafka -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.podIP}{"\n"}{end}'

# Verify Kafka broker connectivity
kubectl exec -n kafka kafka-0 -- kafka-broker-api-versions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092

# List existing topics
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --list
```

---

## Procedure

### Step 1: Assess Current Kafka State

```bash
# Check cluster health and broker count
kubectl exec -n kafka kafka-0 -- kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --cluster-id $(kubectl exec -n kafka kafka-0 -- cat /var/lib/kafka/data/meta.properties | grep cluster.id | cut -d= -f2) 2>/dev/null || \
kubectl exec -n kafka kafka-0 -- kafka-broker-api-versions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092

# Describe all topics with partition/replica details
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe

# Check consumer group lag for all groups
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --all-groups \
  --describe

# Check broker disk usage
for i in 0 1 2; do
  echo "=== kafka-${i} ==="
  kubectl exec -n kafka kafka-${i} -- df -h /var/lib/kafka/data
done

# Check broker log segment sizes per topic
kubectl exec -n kafka kafka-0 -- du -sh /var/lib/kafka/data/clario360.*
```

### Step 2: Add New Kafka Broker to Cluster

#### 2a: Scale the Kafka StatefulSet

```bash
# Check current replica count
kubectl get statefulset kafka -n kafka

# Scale from 3 to 5 brokers
kubectl scale statefulset kafka -n kafka --replicas=5

# Wait for new brokers to become ready
kubectl rollout status statefulset/kafka -n kafka

# Verify all brokers are running
kubectl get pods -n kafka -l app=kafka -o wide

# Confirm new brokers have joined the cluster
kubectl exec -n kafka kafka-0 -- kafka-broker-api-versions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 2>&1 | grep -c "ApiVersion"
```

#### 2b: Verify New Broker Storage

```bash
# Ensure PVCs were created for new brokers
kubectl get pvc -n kafka -l app=kafka

# Verify disk is available on new brokers
for i in 3 4; do
  echo "=== kafka-${i} ==="
  kubectl exec -n kafka kafka-${i} -- df -h /var/lib/kafka/data
done
```

### Step 3: Rebalance Partitions Across Brokers

New brokers do not automatically receive existing partitions. You must manually reassign partitions.

#### 3a: Generate Reassignment Plan

```bash
# Create a JSON file listing topics to rebalance
kubectl exec -n kafka kafka-0 -- bash -c 'cat > /tmp/topics-to-reassign.json << INNER_EOF
{
  "version": 1,
  "topics": [
    {"topic": "clario360.audit.events"},
    {"topic": "clario360.notifications"},
    {"topic": "clario360.workflow.commands"},
    {"topic": "clario360.workflow.events"},
    {"topic": "clario360.cyber.findings"},
    {"topic": "clario360.data.ingestion"},
    {"topic": "clario360.lex.regulatory-updates"},
    {"topic": "clario360.acta.document-events"},
    {"topic": "clario360.visus.report-requests"},
    {"topic": "clario360.events"}
  ]
}
INNER_EOF'

# Generate the reassignment plan (brokers 0-4 after scaling to 5)
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topics-to-move-json-file /tmp/topics-to-reassign.json \
  --broker-list "0,1,2,3,4" \
  --generate

# The output will show:
# 1. Current partition assignment (save as rollback)
# 2. Proposed partition reassignment (save as the plan to execute)

# Save the proposed reassignment to a file
kubectl exec -n kafka kafka-0 -- bash -c 'cat > /tmp/reassignment-plan.json << INNER_EOF
<PASTE_PROPOSED_ASSIGNMENT_JSON_HERE>
INNER_EOF'
```

#### 3b: Execute Reassignment

```bash
# Execute the reassignment plan
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/reassignment-plan.json \
  --execute

# Throttle replication to avoid overwhelming the cluster (100 MB/s per broker)
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/reassignment-plan.json \
  --execute \
  --throttle 104857600
```

#### 3c: Monitor Reassignment Progress

```bash
# Check reassignment status
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/reassignment-plan.json \
  --verify

# Run repeatedly until all partitions show "completed successfully"
# Watch the reassignment in a loop
for i in $(seq 1 60); do
  echo "=== Check ${i} at $(date) ==="
  kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
    --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
    --reassignment-json-file /tmp/reassignment-plan.json \
    --verify 2>&1 | grep -E "Status|completed|in progress"
  sleep 30
done
```

#### 3d: Remove Throttle After Completion

```bash
# Verify all reassignments are complete, then remove throttle
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/reassignment-plan.json \
  --verify

# Confirm throttle is removed (output should mention throttle removal)
```

### Step 4: Increase Partition Count for High-Throughput Topics

Increasing partition count allows more parallelism for consumers. Note: this operation cannot be reversed.

```bash
# Increase partitions for high-throughput topics

# clario360.audit.events: 12 -> 24 partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --topic clario360.audit.events \
  --partitions 24

# clario360.cyber.findings: 12 -> 24 partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --topic clario360.cyber.findings \
  --partitions 24

# clario360.data.ingestion: 12 -> 24 partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --topic clario360.data.ingestion \
  --partitions 24

# clario360.workflow.events: 12 -> 24 partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --topic clario360.workflow.events \
  --partitions 24

# clario360.events: 12 -> 24 partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --topic clario360.events \
  --partitions 24

# Verify the updated partition counts
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --topic clario360.audit.events

kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --topic clario360.cyber.findings
```

**Important:** After increasing partitions, consumer group instances must be restarted to pick up the new partitions.

```bash
# Restart consumer services to pick up new partitions
kubectl rollout restart deployment/audit-service -n clario360
kubectl rollout restart deployment/notification-service -n clario360
kubectl rollout restart deployment/data-service -n clario360
kubectl rollout restart deployment/workflow-engine -n clario360

# Verify consumer groups are balanced across new partitions
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --group audit-service-group \
  --describe

kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --group notification-service-group \
  --describe
```

### Step 5: Adjust Replication Factor

Increasing the replication factor improves durability. This uses the same reassignment mechanism.

```bash
# Check current replication factor for a topic
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --topic clario360.audit.events

# To increase replication factor from 3 to 4 for clario360.audit.events,
# generate a reassignment JSON specifying 4 replicas per partition.
# Example for a topic with 12 partitions across 5 brokers:
kubectl exec -n kafka kafka-0 -- bash -c 'cat > /tmp/increase-rf.json << INNER_EOF
{
  "version": 1,
  "partitions": [
    {"topic": "clario360.audit.events", "partition": 0, "replicas": [0,1,2,3]},
    {"topic": "clario360.audit.events", "partition": 1, "replicas": [1,2,3,4]},
    {"topic": "clario360.audit.events", "partition": 2, "replicas": [2,3,4,0]},
    {"topic": "clario360.audit.events", "partition": 3, "replicas": [3,4,0,1]},
    {"topic": "clario360.audit.events", "partition": 4, "replicas": [4,0,1,2]},
    {"topic": "clario360.audit.events", "partition": 5, "replicas": [0,1,2,4]},
    {"topic": "clario360.audit.events", "partition": 6, "replicas": [1,2,3,0]},
    {"topic": "clario360.audit.events", "partition": 7, "replicas": [2,3,4,1]},
    {"topic": "clario360.audit.events", "partition": 8, "replicas": [3,4,0,2]},
    {"topic": "clario360.audit.events", "partition": 9, "replicas": [4,0,1,3]},
    {"topic": "clario360.audit.events", "partition": 10, "replicas": [0,2,3,4]},
    {"topic": "clario360.audit.events", "partition": 11, "replicas": [1,3,4,0]}
  ]
}
INNER_EOF'

# Execute the replication factor increase
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/increase-rf.json \
  --execute \
  --throttle 104857600

# Verify completion
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --reassignment-json-file /tmp/increase-rf.json \
  --verify

# Update topic min.insync.replicas to match new RF
kubectl exec -n kafka kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --entity-type topics \
  --entity-name clario360.audit.events \
  --add-config min.insync.replicas=3
```

### Step 6: Tune Broker Configuration for Scaled Cluster

```bash
# Adjust broker configurations for larger cluster
kubectl exec -n kafka kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --alter \
  --entity-type brokers \
  --entity-default \
  --add-config \
    num.io.threads=16,\
    num.network.threads=8,\
    num.replica.fetchers=4,\
    replica.fetch.max.bytes=10485760,\
    socket.send.buffer.bytes=1048576,\
    socket.receive.buffer.bytes=1048576,\
    log.retention.hours=168,\
    log.segment.bytes=1073741824,\
    log.retention.check.interval.ms=300000

# Verify configuration
kubectl exec -n kafka kafka-0 -- kafka-configs.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --entity-type brokers \
  --entity-default
```

### Step 7: Monitor Broker Performance After Scaling

```bash
# Check under-replicated partitions (should be 0 when healthy)
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --under-replicated-partitions

# Check offline partitions (should be 0)
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --describe \
  --unavailable-partitions

# Check consumer group lag across all groups
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --all-groups \
  --describe 2>&1 | grep -v "^$"

# Check broker JMX metrics via Prometheus (if JMX exporter is deployed)
# Key metrics to monitor:
# - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec
# - kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec
# - kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec
# - kafka.server:type=ReplicaManager,name=UnderReplicatedPartitions
# - kafka.server:type=ReplicaManager,name=IsrShrinksPerSec
# - kafka.server:type=ReplicaManager,name=IsrExpandsPerSec

# Check broker disk usage after rebalance
for i in 0 1 2 3 4; do
  echo "=== kafka-${i} ==="
  kubectl exec -n kafka kafka-${i} -- df -h /var/lib/kafka/data 2>/dev/null || echo "Broker ${i} not available"
done

# Check broker CPU and memory usage
kubectl top pods -n kafka -l app=kafka

# Produce and consume a test message to verify end-to-end functionality
kubectl exec -n kafka kafka-0 -- bash -c '
  echo "scaling-test-$(date +%s)" | kafka-console-producer.sh \
    --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
    --topic clario360.events
'

kubectl exec -n kafka kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
  --topic clario360.events \
  --from-latest \
  --max-messages 1 \
  --timeout-ms 10000
```

---

## Verification

After completing Kafka scaling, confirm the following:

1. All brokers are online and in the cluster:
   ```bash
   kubectl get pods -n kafka -l app=kafka -o wide
   ```

2. No under-replicated partitions exist:
   ```bash
   kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
     --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
     --describe --under-replicated-partitions
   ```

3. Consumer group lag is decreasing or stable:
   ```bash
   kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
     --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
     --all-groups --describe
   ```

4. Partition distribution is balanced across brokers:
   ```bash
   kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
     --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
     --describe | awk -F'\t' '/Partition:/{print $NF}' | sort | uniq -c | sort -rn
   ```

5. End-to-end produce/consume works:
   ```bash
   kubectl exec -n kafka kafka-0 -- bash -c '
     echo "verify-$(date +%s)" | kafka-console-producer.sh \
       --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092 \
       --topic clario360.events
   '
   ```

---

## Related Links

| Document                  | Link                                             |
|---------------------------|--------------------------------------------------|
| Horizontal Pod Scaling    | [SC-001-horizontal-scaling.md](SC-001-horizontal-scaling.md) |
| Database Scaling          | [SC-002-database-scaling.md](SC-002-database-scaling.md)     |
| Node Pool Scaling         | [SC-004-node-pool-scaling.md](SC-004-node-pool-scaling.md)   |
| Capacity Planning         | [SC-005-capacity-planning.md](SC-005-capacity-planning.md)   |
| Kafka Operations Docs     | https://kafka.apache.org/documentation/#operations           |
