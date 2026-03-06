package tracing

import "go.opentelemetry.io/otel/attribute"

// Standard attribute keys used across all Clario 360 services.
// Centralizing these prevents typos and inconsistent naming.
const (
	AttrTenantID       = attribute.Key("clario.tenant_id")
	AttrUserID         = attribute.Key("clario.user_id")
	AttrRequestID      = attribute.Key("clario.request_id")
	AttrCorrelationID  = attribute.Key("clario.correlation_id")
	AttrService        = attribute.Key("clario.service")
	AttrAction         = attribute.Key("clario.action")
	AttrResourceType   = attribute.Key("clario.resource_type")
	AttrResourceID     = attribute.Key("clario.resource_id")
	AttrKafkaTopic     = attribute.Key("messaging.kafka.topic")
	AttrKafkaGroup     = attribute.Key("messaging.kafka.consumer_group")
	AttrKafkaPartition = attribute.Key("messaging.kafka.partition")
	AttrDBOperation    = attribute.Key("db.operation")
	AttrDBStatement    = attribute.Key("db.statement")
	AttrDBSystem       = attribute.Key("db.system")
	AttrDBTable        = attribute.Key("db.sql.table")
)
