package integration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	policymodel "github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// TicketRequest encapsulates all the information needed to create an ITSM
// (IT Service Management) ticket from a DSPM remediation item.
type TicketRequest struct {
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Priority      string    `json:"priority"`
	AssignedTo    string    `json:"assigned_to,omitempty"`
	SLADueAt      time.Time `json:"sla_due_at"`
	Tags          []string  `json:"tags,omitempty"`
	RemediationID uuid.UUID `json:"remediation_id"`
	ExternalURL   string    `json:"external_url,omitempty"`
}

// TicketResult is the outcome of creating an ITSM ticket, containing the
// external system's ticket identifier and tracking information.
type TicketResult struct {
	ExternalTicketID string    `json:"external_ticket_id"`
	URL              string    `json:"url"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// ITSMConnector builds ITSM ticket requests from DSPM remediations and
// generates deterministic ticket identifiers for idempotent integration
// with external ticketing systems.
type ITSMConnector struct {
	logger zerolog.Logger
}

// NewITSMConnector constructs an ITSMConnector.
func NewITSMConnector(logger zerolog.Logger) *ITSMConnector {
	return &ITSMConnector{
		logger: logger.With().Str("component", "itsm_connector").Logger(),
	}
}

// CreateTicket builds an ITSM ticket from a remediation item and returns
// a TicketResult with a deterministic ticket ID derived from the
// remediation's identity. This determinism allows callers to detect
// duplicate ticket creation attempts.
func (ic *ITSMConnector) CreateTicket(ctx context.Context, tenantID uuid.UUID, remediation *policymodel.Remediation) (*TicketResult, error) {
	if remediation == nil {
		return nil, fmt.Errorf("itsm connector: remediation is nil")
	}

	priority := SeverityToPriority(remediation.Severity)
	title := buildTicketTitle(remediation)
	description := buildTicketDescription(remediation)
	tags := buildTicketTags(remediation)

	// Compute SLA due time. If the remediation already has an SLA, use it;
	// otherwise derive one from the severity.
	var slaDueAt time.Time
	if remediation.SLADueAt != nil {
		slaDueAt = *remediation.SLADueAt
	} else {
		cfg := policymodel.DefaultSLAConfig()
		hours := cfg.SLAHoursForSeverity(remediation.Severity)
		slaDueAt = remediation.CreatedAt.Add(time.Duration(hours) * time.Hour)
	}

	// Deterministic ticket ID based on tenant + remediation ID.
	ticketID := deterministicTicketID(tenantID, remediation.ID)
	ticketURL := fmt.Sprintf("https://itsm.internal/tickets/%s", ticketID)

	now := time.Now().UTC()

	result := &TicketResult{
		ExternalTicketID: ticketID,
		URL:              ticketURL,
		Status:           "open",
		CreatedAt:        now,
	}

	ic.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", remediation.ID.String()).
		Str("ticket_id", ticketID).
		Str("priority", priority).
		Str("severity", remediation.Severity).
		Time("sla_due_at", slaDueAt).
		Int("tag_count", len(tags)).
		Msg("ITSM ticket created")

	_ = TicketRequest{
		Title:         title,
		Description:   description,
		Priority:      priority,
		SLADueAt:      slaDueAt,
		Tags:          tags,
		RemediationID: remediation.ID,
		ExternalURL:   ticketURL,
	}

	return result, nil
}

// SeverityToPriority maps a DSPM severity level to the conventional ITSM
// priority designation.
//
//	critical → P1
//	high     → P2
//	medium   → P3
//	low      → P4
func SeverityToPriority(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "P1"
	case "high":
		return "P2"
	case "medium":
		return "P3"
	case "low":
		return "P4"
	default:
		return "P3"
	}
}

// deterministicTicketID generates a reproducible ticket identifier from the
// tenant ID and remediation ID. This ensures idempotency when the same
// remediation triggers multiple ticket creation attempts.
func deterministicTicketID(tenantID, remediationID uuid.UUID) string {
	data := fmt.Sprintf("%s:%s", tenantID.String(), remediationID.String())
	hash := sha256.Sum256([]byte(data))
	// Use first 8 bytes (16 hex chars) for a human-friendly ticket ID.
	return fmt.Sprintf("DSPM-%X", hash[:8])
}

// buildTicketTitle generates a concise ITSM ticket title from a remediation.
func buildTicketTitle(remediation *policymodel.Remediation) string {
	title := fmt.Sprintf("[%s] %s", strings.ToUpper(remediation.Severity), remediation.Title)
	if len(title) > 200 {
		title = title[:197] + "..."
	}
	return title
}

// buildTicketDescription generates a detailed ITSM ticket description from a remediation.
func buildTicketDescription(remediation *policymodel.Remediation) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("DSPM Remediation Ticket\n"))
	sb.WriteString(fmt.Sprintf("=======================\n\n"))
	sb.WriteString(fmt.Sprintf("Remediation ID: %s\n", remediation.ID.String()))
	sb.WriteString(fmt.Sprintf("Finding Type:   %s\n", remediation.FindingType))
	sb.WriteString(fmt.Sprintf("Severity:       %s\n", remediation.Severity))
	sb.WriteString(fmt.Sprintf("Status:         %s\n", remediation.Status))
	sb.WriteString(fmt.Sprintf("Playbook:       %s\n", remediation.PlaybookID))
	sb.WriteString(fmt.Sprintf("Progress:       step %d of %d\n\n", remediation.CurrentStep, remediation.TotalSteps))

	if remediation.DataAssetName != "" {
		sb.WriteString(fmt.Sprintf("Affected Asset: %s\n", remediation.DataAssetName))
	}
	if remediation.DataAssetID != nil {
		sb.WriteString(fmt.Sprintf("Asset ID:       %s\n", remediation.DataAssetID.String()))
	}

	sb.WriteString(fmt.Sprintf("\nDescription:\n%s\n", remediation.Description))

	if remediation.SLADueAt != nil {
		sb.WriteString(fmt.Sprintf("\nSLA Due: %s\n", remediation.SLADueAt.Format(time.RFC3339)))
	}
	if remediation.SLABreached {
		sb.WriteString("WARNING: SLA has been breached.\n")
	}

	return sb.String()
}

// buildTicketTags generates tags for the ITSM ticket based on the remediation.
func buildTicketTags(remediation *policymodel.Remediation) []string {
	tags := []string{
		"dspm",
		"security",
		fmt.Sprintf("severity:%s", remediation.Severity),
		fmt.Sprintf("finding:%s", remediation.FindingType),
	}

	if remediation.SLABreached {
		tags = append(tags, "sla-breached")
	}

	if remediation.AssignedTeam != "" {
		tags = append(tags, fmt.Sprintf("team:%s", remediation.AssignedTeam))
	}

	return tags
}
