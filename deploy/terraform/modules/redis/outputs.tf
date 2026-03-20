output "host" {
  description = "Redis instance host IP"
  value       = google_redis_instance.main.host
}

output "port" {
  description = "Redis instance port"
  value       = google_redis_instance.main.port
}

output "connection_string" {
  description = "Redis connection string (redis://host:port)"
  value       = "redis://${google_redis_instance.main.host}:${google_redis_instance.main.port}"
}

output "instance_id" {
  description = "Redis instance ID"
  value       = google_redis_instance.main.id
}

output "read_endpoint" {
  description = "Redis read endpoint (available when replicas are enabled)"
  value       = google_redis_instance.main.read_endpoint
}
