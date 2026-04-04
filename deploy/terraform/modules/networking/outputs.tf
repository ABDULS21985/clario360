output "vpc_id" {
  description = "The ID of the VPC"
  value       = google_compute_network.vpc.id
}

output "vpc_name" {
  description = "The name of the VPC"
  value       = google_compute_network.vpc.name
}

output "vpc_self_link" {
  description = "The self link of the VPC"
  value       = google_compute_network.vpc.self_link
}

output "public_subnet_id" {
  description = "The ID of the public subnet"
  value       = google_compute_subnetwork.public.id
}

output "public_subnet_name" {
  description = "The name of the public subnet"
  value       = google_compute_subnetwork.public.name
}

output "private_subnet_id" {
  description = "The ID of the private subnet"
  value       = google_compute_subnetwork.private.id
}

output "private_subnet_name" {
  description = "The name of the private subnet"
  value       = google_compute_subnetwork.private.name
}

output "isolated_subnet_id" {
  description = "The ID of the isolated subnet"
  value       = google_compute_subnetwork.isolated.id
}

output "isolated_subnet_name" {
  description = "The name of the isolated subnet"
  value       = google_compute_subnetwork.isolated.name
}

output "pods_secondary_range_name" {
  description = "The name of the secondary range for GKE pods"
  value       = google_compute_subnetwork.private.secondary_ip_range[0].range_name
}

output "services_secondary_range_name" {
  description = "The name of the secondary range for GKE services"
  value       = google_compute_subnetwork.private.secondary_ip_range[1].range_name
}

output "nat_ip_addresses" {
  description = "The NAT IP addresses"
  value       = google_compute_router_nat.nat.nat_ips
}

output "psa_connection_id" {
  description = "The private service access connection ID (for Cloud SQL dependency)"
  value       = google_service_networking_connection.psa.id
}

output "router_name" {
  description = "The name of the Cloud Router"
  value       = google_compute_router.router.name
}
