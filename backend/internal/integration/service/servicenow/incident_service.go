package servicenow

import (
	"context"
	"fmt"
	"strings"
	"time"

	intmodel "github.com/clario360/platform/internal/integration/model"
)

type ClarioClient interface {
	FetchEntity(ctx context.Context, token, entityType, entityID string) (map[string]any, error)
	MintSystemToken(tenantID string) (string, error)
	UpdateAlertStatus(ctx context.Context, token, alertID, status string, notes, reason *string) error
	AddAlertComment(ctx context.Context, token, alertID, content string, metadata map[string]any) error
}

type TicketLinkStore interface {
	Create(ctx context.Context, link *intmodel.ExternalTicketLink) (string, error)
	GetByExternal(ctx context.Context, externalSystem, externalID string) (*intmodel.ExternalTicketLink, error)
	UpdateSync(ctx context.Context, link *intmodel.ExternalTicketLink) error
}

type Service struct {
	client *Client
	clario ClarioClient
	links  TicketLinkStore
	appURL string
}

func NewIncidentService(client *Client, clario ClarioClient, links TicketLinkStore, appURL string) *Service {
	return &Service{client: client, clario: clario, links: links, appURL: strings.TrimRight(appURL, "/")}
}

func (s *Service) RawClient() *Client {
	return s.client
}

func (s *Service) ClarioToken(ctx context.Context, tenantID string) (string, error) {
	return s.clario.MintSystemToken(tenantID)
}

func (s *Service) CreateFromEntity(ctx context.Context, integration *intmodel.Integration, cfg intmodel.ServiceNowConfig, bearerToken, entityType, entityID string) (*intmodel.ExternalTicketLink, int, string, error) {
	entity, err := s.clario.FetchEntity(ctx, bearerToken, entityType, entityID)
	if err != nil {
		return nil, 0, "", err
	}
	title := firstNonEmpty(stringValue(entity["title"]), entityType+" "+entityID)
	description := firstNonEmpty(extractSummary(entity), stringValue(entity["description"]))
	severity := firstNonEmpty(stringValue(entity["severity"]), stringValue(entity["priority"]))

	payload := map[string]any{
		"short_description": truncate(fmt.Sprintf("[Clario 360] %s", title), 160),
		"description":       description,
		"urgency":           MapUrgency(severity),
		"impact":            MapImpact(1, strings.EqualFold(severity, "critical")),
		"category":          firstNonEmpty(cfg.Category, "Security"),
		"subcategory":       firstNonEmpty(cfg.Subcategory, "Threat Detection"),
		"assignment_group":  cfg.AssignmentGroup,
		"caller_id":         cfg.CallerID,
	}
	for key, value := range cfg.CustomFields {
		payload[key] = value
	}

	incident, code, body, err := s.client.CreateIncident(ctx, cfg, payload)
	if err != nil {
		return nil, code, body, err
	}
	result, _ := incident["result"].(map[string]any)
	sysID := stringValue(result["sys_id"])
	number := stringValue(result["number"])
	link := &intmodel.ExternalTicketLink{
		TenantID:       integration.TenantID,
		IntegrationID:  integration.ID,
		EntityType:     entityType,
		EntityID:       entityID,
		ExternalSystem: "servicenow",
		ExternalID:     sysID,
		ExternalKey:    number,
		ExternalURL:    strings.TrimRight(cfg.InstanceURL, "/") + "/incident.do?sys_id=" + sysID,
		SyncDirection:  intmodel.SyncDirectionBidirectional,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if _, err := s.links.Create(ctx, link); err != nil {
		return nil, code, body, err
	}
	if entityType == "alert" {
		_ = s.clario.AddAlertComment(ctx, bearerToken, entityID, "Linked ServiceNow incident "+number, map[string]any{
			"external_system": "servicenow",
			"external_key":    number,
			"external_url":    link.ExternalURL,
		})
	}
	return link, code, body, nil
}

func (s *Service) SyncWebhookStatus(ctx context.Context, integration *intmodel.Integration, cfg intmodel.ServiceNowConfig, externalID, externalStatus string) (*intmodel.ExternalTicketLink, error) {
	link, err := s.links.GetByExternal(ctx, "servicenow", externalID)
	if err != nil {
		return nil, err
	}
	mappedStatus := MapStateToClario(cfg.StatusMapping, externalStatus)
	now := time.Now().UTC()
	link.ExternalStatus = &externalStatus
	link.LastSyncedAt = &now
	dir := "inbound"
	link.LastSyncDirection = &dir
	if mappedStatus == "" || link.EntityType != "alert" {
		return link, s.links.UpdateSync(ctx, link)
	}
	systemToken, err := s.clario.MintSystemToken(link.TenantID)
	if err != nil {
		return nil, err
	}
	note := fmt.Sprintf("Status updated from ServiceNow (%s): %s", link.ExternalKey, externalStatus)
	if err := s.clario.UpdateAlertStatus(ctx, systemToken, link.EntityID, mappedStatus, &note, nil); err != nil {
		errText := err.Error()
		link.SyncError = &errText
		_ = s.links.UpdateSync(ctx, link)
		return nil, err
	}
	_ = s.clario.AddAlertComment(ctx, systemToken, link.EntityID, note, map[string]any{
		"external_system": "servicenow",
		"external_key":    link.ExternalKey,
		"external_status": externalStatus,
	})
	link.SyncError = nil
	return link, s.links.UpdateSync(ctx, link)
}

func (s *Service) SyncLink(ctx context.Context, integration *intmodel.Integration, cfg intmodel.ServiceNowConfig, link *intmodel.ExternalTicketLink) error {
	incident, _, _, err := s.client.GetIncident(ctx, cfg, link.ExternalID)
	if err != nil {
		return err
	}
	result, _ := incident["result"].(map[string]any)
	state := stringValue(result["state"])
	_, err = s.SyncWebhookStatus(ctx, integration, cfg, link.ExternalID, state)
	return err
}

func extractSummary(entity map[string]any) string {
	if explanation, ok := entity["explanation"].(map[string]any); ok {
		return stringValue(explanation["summary"])
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
