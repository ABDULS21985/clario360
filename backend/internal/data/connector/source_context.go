package connector

import "github.com/google/uuid"

type SourceContextAware interface {
	SetSourceContext(sourceID, tenantID uuid.UUID)
}
