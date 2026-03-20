# IR-003: Kafka Broker Failure

| Field              | Value                                                                 |
|--------------------|-----------------------------------------------------------------------|
| **Runbook ID**     | IR-003                                                                |
| **Title**          | Kafka Broker Failure                                                  |
| **Severity**       | P1 -- Critical                                                        |
| **Author**         | Clario360 Platform Team                                               |
| **Last Updated**   | 2026-03-08                                                            |
| **Review Cycle**   | Quarterly                                                             |
| **Applies To**     | Kafka brokers in namespace `kafka`, Clario360 producers and consumers |
| **Namespace**      | kafka (brokers), clario360 (application services)                     |
| **Broker Pod**     | kafka-0 (StatefulSet `kafka` in namespace `kafka`)                    |
| **Escalation**     | Platform Engineering Lead -> VP Engineering -> CTO                    |
| **SLA**            | Acknowledge within 5 minutes, resolve within 45 minutes               |

---

## Summary

This runbook covers incidents related to Kafka broker failures affecting the Clario360 platform. Scenarios include one or more brokers going down, under-replicated partitions causing data durability risks, and consumer group lag spikes indicating consumers are unable to keep up with producers. Kafka is used for asynchronous communication between Clario360 services including audit events, notifications, workflow events, and data pipeline messages.

---

## Symptoms

- Application logs showing `kafka: client has run out of available brokers`, `kafka: request timeout`, or `kafka: connection refused`.
- Audit events, notifications, or workflow transitions not being processed.
- Consumer group lag increasing steadily on Grafana Kafka dashboards.
- Kafka broker pods in `CrashLoopBackOff`, `Error`, or `NotReady` state.
- Under-replicated partition alerts from Kafka JMX metrics.
- Producer services logging `kafka: message too large` or `kafka: not enough replicas`.
- Disk usage alerts on Kafka broker PersistentVolumes.

---

## Impact Assessment

| Kafka Topic Pattern           | Producing Services                     | Consuming Services                    | Business Impact                                      |
|-------------------------------|----------------------------------------|---------------------------------------|------------------------------------------------------|
| `audit.*`                     | All services                           | audit-service                         | Audit log ingestion stops; compliance gap            |
| `notifications.*`            | workflow-engine, iam-service           | notification-service                  | Notifications delayed or lost                        |
| `workflow.*`                 | api-gateway, workflow-engine           | workflow-engine                       | Workflow state transitions stall                     |
| `cyber.*`                    | cyber-service, data-service            | cyber-service                         | Security event processing halted                     |
| `data.*`                     | data-service                           | data-service, visus-service           | Data pipeline stalls; reports stale                  |
| `events`                     | All services                           | Multiple consumers                    | Cross-service event propagation stops                |

---

## Prerequisites

- `kubectl` configured with access to both `kafka` and `clario360` namespaces.
- Familiarity with Kafka CLI tools (`kafka-topics.sh`, `kafka-consumer-groups.sh`).
- Access to Grafana Kafka dashboards.
- Knowledge of the Kafka cluster topology (number of brokers, replication factor).

---

## Diagnosis Steps

### Step 1: Check Kafka Broker Pod Status

```bash
kubectl get pods -n kafka -l app=kafka -o wide
```

```bash
kubectl get pods -n kafka -l app=kafka -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\t"}{.status.containerStatuses[0].ready}{"\t"}{.status.containerStatuses[0].restartCount}{"\n"}{end}'
```

### Step 2: Check Kafka Broker Logs

```bash
kubectl logs -n kafka kafka-0 --tail=300 --timestamps
```

Look for:
- `WARN` or `ERROR` entries related to `ISR shrink`, `UnderReplicatedPartitions`, `OutOfMemoryError`.
- `IOException` related to disk writes.
- `ControllerMovedException` or `NotLeaderForPartitionException`.
- `Log segment deletion` messages if disk is full.

### Step 3: Check Kafka Broker Events

```bash
kubectl describe pod kafka-0 -n kafka
```

### Step 4: Check Zookeeper Status (If Using ZK Mode)

```bash
kubectl get pods -n kafka -l app=zookeeper -o wide
kubectl logs -n kafka -l app=zookeeper --tail=100 --timestamps
```

### Step 5: List All Topics and Check for Under-Replicated Partitions

```bash
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --under-replicated-partitions
```

### Step 6: List All Topics with Partition Details

```bash
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list
```

```bash
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic audit.events
```

### Step 7: Check Consumer Group Lag

```bash
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --list
```

```bash
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group audit-service-group
```

```bash
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group notification-service-group
```

```bash
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group workflow-engine-group
```

### Step 8: Check Broker Disk Usage

```bash
kubectl exec -n kafka kafka-0 -- df -h /var/lib/kafka/data
```

```bash
kubectl exec -n kafka kafka-0 -- du -sh /var/lib/kafka/data/*
```

### Step 9: Check Broker Resource Usage

```bash
kubectl top pod -n kafka -l app=kafka
```

### Step 10: Verify Broker Connectivity from Application Namespace

```bash
kubectl run kafka-debug --rm -it --restart=Never -n clario360 \
  --image=confluentinc/cp-kafka:7.5.0 \
  -- kafka-broker-api-versions.sh --bootstrap-server kafka-0.kafka-headless.kafka.svc.cluster.local:9092
```

---

## Resolution Steps

### Scenario A: Single Broker Down

**1. Check why the broker is down (see Diagnosis Step 2 and 3).**

**2. Restart the broker pod:**

```bash
kubectl delete pod kafka-0 -n kafka
```

The StatefulSet controller will recreate it. Wait for it to rejoin the cluster:

```bash
kubectl rollout status statefulset/kafka -n kafka --timeout=300s
```

**3. Verify the broker has rejoined the cluster and partitions are in-sync:**

```bash
kubectl exec -n kafka kafka-0 -- kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --cluster-id $(kubectl exec -n kafka kafka-0 -- cat /var/lib/kafka/data/meta.properties 2>/dev/null | grep cluster.id | cut -d= -f2) 2>/dev/null || \
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --under-replicated-partitions
```

### Scenario B: Under-Replicated Partitions

**1. Identify which partitions are under-replicated:**

```bash
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --under-replicated-partitions
```

**2. If a broker is lagging, restart it to trigger a full resync:**

```bash
kubectl delete pod kafka-<BROKER_ID> -n kafka
```

**3. If partitions need to be reassigned to available brokers, generate a reassignment plan:**

```bash
kubectl exec -n kafka kafka-0 -- bash -c 'cat > /tmp/topics.json << INNEREOF
{"topics": [{"topic": "audit.events"}, {"topic": "notifications.events"}, {"topic": "workflow.events"}], "version": 1}
INNEREOF
kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --topics-to-move-json-file /tmp/topics.json \
  --broker-list "0,1,2" \
  --generate'
```

**4. Execute the reassignment (use the output from the previous command):**

```bash
kubectl exec -n kafka kafka-0 -- bash -c 'cat > /tmp/reassignment.json << INNEREOF
<PASTE_PROPOSED_REASSIGNMENT_JSON_HERE>
INNEREOF
kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --execute'
```

**5. Verify reassignment progress:**

```bash
kubectl exec -n kafka kafka-0 -- kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --verify
```

### Scenario C: Consumer Group Lag Spike

**1. Check if the consumer service is running and healthy:**

```bash
kubectl get pods -n clario360 -l app=<CONSUMER_SERVICE>
kubectl logs -n clario360 -l app=<CONSUMER_SERVICE> --tail=200 --timestamps | grep -i -E "kafka|consumer|error|timeout"
```

**2. Restart the consumer service to reset connections:**

```bash
kubectl rollout restart deployment/<CONSUMER_SERVICE> -n clario360
kubectl rollout status deployment/<CONSUMER_SERVICE> -n clario360 --timeout=120s
```

**3. Scale up the consumer to increase throughput (ensure partition count >= replica count):**

```bash
kubectl scale deployment/<CONSUMER_SERVICE> -n clario360 --replicas=5
```

**4. If the consumer needs to skip corrupted messages, reset offsets to latest (data loss -- use only as last resort):**

```bash
# First, scale down the consumer
kubectl scale deployment/<CONSUMER_SERVICE> -n clario360 --replicas=0

# Reset offsets to latest
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group <CONSUMER_GROUP_NAME> \
  --topic <TOPIC_NAME> \
  --reset-offsets \
  --to-latest \
  --execute

# Scale consumer back up
kubectl scale deployment/<CONSUMER_SERVICE> -n clario360 --replicas=3
```

### Scenario D: Kafka Disk Full

**1. Reduce retention for high-volume topics temporarily:**

```bash
kubectl exec -n kafka kafka-0 -- kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type topics \
  --entity-name audit.events \
  --alter \
  --add-config retention.ms=86400000
```

(Sets retention to 24 hours. Adjust topic name and retention as needed.)

**2. Trigger log segment deletion:**

```bash
kubectl exec -n kafka kafka-0 -- kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type topics \
  --entity-name audit.events \
  --alter \
  --add-config retention.bytes=5368709120
```

(Sets retention to 5 GB per partition.)

**3. Expand the Kafka PersistentVolumeClaim (if storage class supports expansion):**

```bash
kubectl patch pvc data-kafka-0 -n kafka -p '{"spec": {"resources": {"requests": {"storage": "100Gi"}}}}'
```

**4. Verify disk usage has decreased:**

```bash
kubectl exec -n kafka kafka-0 -- df -h /var/lib/kafka/data
```

---

## Verification

```bash
# 1. All broker pods are running and ready
kubectl get pods -n kafka -l app=kafka -o wide

# 2. No under-replicated partitions
kubectl exec -n kafka kafka-0 -- kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --under-replicated-partitions

# 3. Consumer group lag is decreasing or zero
kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group audit-service-group

kubectl exec -n kafka kafka-0 -- kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group notification-service-group

# 4. Produce and consume a test message
kubectl exec -n kafka kafka-0 -- bash -c 'echo "test-$(date +%s)" | kafka-console-producer.sh --bootstrap-server localhost:9092 --topic _healthcheck'

kubectl exec -n kafka kafka-0 -- kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic _healthcheck \
  --from-beginning \
  --max-messages 1 \
  --timeout-ms 10000

# 5. Application services are healthy
for svc in audit-service notification-service workflow-engine; do
  echo -n "$svc readyz: "
  kubectl exec -n clario360 $(kubectl get pod -n clario360 -l app=$svc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- wget -q -O- http://localhost:8080/readyz 2>/dev/null && echo " OK" || echo " FAIL"
done

# 6. Disk usage is below 80%
kubectl exec -n kafka kafka-0 -- df -h /var/lib/kafka/data
```

---

## Post-Incident Checklist

- [ ] Confirm all Kafka broker pods are `Running` and `Ready`.
- [ ] Confirm zero under-replicated partitions.
- [ ] Confirm consumer group lag is returning to normal levels.
- [ ] Confirm application services can produce and consume messages.
- [ ] Verify Prometheus/Grafana alerts have cleared.
- [ ] Check if any messages were lost during the incident.
- [ ] Notify stakeholders of resolution.
- [ ] Create post-incident review (PIR) ticket.
- [ ] Document root cause and corrective actions.
- [ ] Review replication factor for critical topics (should be >= 2).
- [ ] Review disk retention policies if disk was the root cause.
- [ ] Consider adding topic-level monitoring for under-replicated partitions.
- [ ] If consumer lag was the issue, review consumer scaling strategy and partition count.

---

## Related Links

| Resource                        | Link                                                         |
|---------------------------------|--------------------------------------------------------------|
| Kafka Operations Guide          | https://kafka.apache.org/documentation/#operations           |
| Grafana Kafka Dashboard         | https://grafana.clario360.internal/d/kafka                    |
| Kafka Topic Configuration       | https://kafka.apache.org/documentation/#topicconfigs         |
| IR-001 Service Outage           | [IR-001-service-outage.md](./IR-001-service-outage.md)       |
| IR-002 Database Failure         | [IR-002-database-failure.md](./IR-002-database-failure.md)   |
| IR-004 Redis Failure            | [IR-004-redis-failure.md](./IR-004-redis-failure.md)         |
| IR-005 Certificate Expiry       | [IR-005-certificate-expiry.md](./IR-005-certificate-expiry.md) |
