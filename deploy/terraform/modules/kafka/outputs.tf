output "cluster_name" {
  description = "Kafka cluster name"
  value       = "clario360"
}

output "bootstrap_servers" {
  description = "Kafka bootstrap servers address"
  value       = "clario360-kafka-bootstrap.${var.namespace}.svc.cluster.local:9092"
}

output "bootstrap_servers_tls" {
  description = "Kafka TLS bootstrap servers address (staging/production)"
  value       = var.environment != "dev" ? "clario360-kafka-bootstrap.${var.namespace}.svc.cluster.local:9093" : ""
}

output "namespace" {
  description = "Kafka namespace"
  value       = var.namespace
}

output "topic_names" {
  description = "List of all Kafka topic names"
  value       = [for name, _ in local.kafka_topics : name]
}

output "dlq_topic_names" {
  description = "List of all DLQ topic names"
  value       = [for name, _ in local.kafka_topics : "${name}.dlq"]
}
