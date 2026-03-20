variable "environment" {
  type        = string
  description = "Environment name"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "namespace" {
  type        = string
  description = "Kubernetes namespace for Kafka"
  default     = "kafka"
}

variable "kafka_replicas" {
  type        = number
  description = "Number of Kafka broker replicas"
  default     = 1

  validation {
    condition     = contains([1, 3], var.kafka_replicas)
    error_message = "Kafka replicas must be 1 (dev) or 3 (staging/production)."
  }
}

variable "kafka_storage_size" {
  type        = string
  description = "Persistent storage size per Kafka broker"
  default     = "10Gi"
}

variable "kafka_memory_request" {
  type        = string
  description = "Memory request per Kafka broker"
  default     = "1Gi"
}

variable "kafka_memory_limit" {
  type        = string
  description = "Memory limit per Kafka broker"
  default     = "2Gi"
}

variable "kafka_cpu_request" {
  type        = string
  description = "CPU request per Kafka broker"
  default     = "500m"
}

variable "kafka_cpu_limit" {
  type        = string
  description = "CPU limit per Kafka broker"
  default     = "1000m"
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
