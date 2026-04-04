#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Create CTI Kafka Topics
# =============================================================================
set -euo pipefail

KAFKA_BROKER="${KAFKA_BROKER:-localhost:9094}"

topics=(
  "cyber.cti.threat-events:6:30"
  "cyber.cti.campaigns:3:30"
  "cyber.cti.brand-abuse:3:30"
  "cyber.cti.feed-ingestion:6:7"
  "cyber.cti.aggregation-triggers:1:1"
  "cyber.cti.alerts:3:14"
  "cyber.cti.dlq:1:90"
)

echo "Creating CTI Kafka topics on ${KAFKA_BROKER}..."

for topic_config in "${topics[@]}"; do
  IFS=: read -r topic partitions retention_days <<< "$topic_config"
  retention_ms=$((retention_days * 86400000))
  kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
    --create --if-not-exists \
    --topic "$topic" \
    --partitions "$partitions" \
    --config "retention.ms=${retention_ms}" 2>/dev/null || \
  echo "  (topic may already exist: $topic)"
  echo "  ✓ $topic (partitions=$partitions, retention=${retention_days}d)"
done

echo ""
echo "CTI topics:"
kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list 2>/dev/null | grep "cyber.cti" || true
echo "Done."
