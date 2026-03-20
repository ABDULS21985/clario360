output "zone_name" {
  description = "Cloud DNS zone name"
  value       = var.create_zone ? google_dns_managed_zone.main[0].name : var.dns_zone_name
}

output "name_servers" {
  description = "Name servers for the DNS zone"
  value       = var.create_zone ? google_dns_managed_zone.main[0].name_servers : []
}

output "dns_records" {
  description = "Map of DNS record names"
  value       = { for k, v in google_dns_record_set.records : k => v.name }
}
