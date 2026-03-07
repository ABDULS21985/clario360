package dto

import (
	"time"

	"github.com/google/uuid"
)

type ComplianceResultsQuery struct {
	CommitteeID *uuid.UUID
	CheckType   *string
	Statuses    []string
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PerPage     int
}
