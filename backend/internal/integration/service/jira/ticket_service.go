package jira

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
	client  *Client
	clario  ClarioClient
	links   TicketLinkStore
	appURL  string
}

func NewTicketService(client *Client, clario ClarioClient, links TicketLinkStore, appURL string) *Service {
	return &Service{client: client, clario: clario, links: links, appURL: strings.TrimRight(appURL, "/")}
}

func (s *Service) RawClient() *Client {
	return s.client
}

func (s *Service) ClarioToken(ctx context.Context, tenantID string) (string, error) {
	return s.clario.MintSystemToken(tenantID)
}

func (s *Service) CreateFromEntity(ctx context.Context, integration *intmodel.Integration, cfg intmodel.JiraConfig, bearerToken, entityType, entityID string) (*intmodel.ExternalTicketLink, int, string, error) {
	entity, err := s.clario.FetchEntity(ctx, bearerToken, entityType, entityID)
	if err != nil {
		return nil, 0, "", err
	}

	title := firstNonEmpty(stringValue(entity["title"]), entityType+" "+entityID)
	severity := firstNonEmpty(stringValue(entity["severity"]), stringValue(entity["priority"]))
	viewURL := s.entityViewURL(entityType, entityID)

	payload := map[string]any{
		"fields": map[string]any{
			"project": map[string]any{"key": cfg.ProjectKey},
			"summary": fmt.Sprintf("[Clario 360 %s] %s", strings.Title(strings.ToLower(severity)), title),
			"description": BuildADF(entity, viewURL),
			"priority": map[string]any{"name": MapSeverityToPriority(cfg, severity)},
			"labels":   []string{"clario360", entityType},
		},
	}
	if cfg.IssueTypeID != "" {
		payload["fields"].(map[string]any)["issuetype"] = map[string]any{"id": cfg.IssueTypeID}
	}
	for key, value := range cfg.CustomFields {
		payload["fields"].(map[string]any)[key] = value
	}

	issue, code, body, err := s.client.CreateIssue(ctx, cfg, payload)
	if err != nil {
		return nil, code, body, err
	}

	externalID := stringValue(issue["id"])
	externalKey := stringValue(issue["key"])
	link := &intmodel.ExternalTicketLink{
		TenantID:      integration.TenantID,
		IntegrationID: integration.ID,
		EntityType:    entityType,
		EntityID:      entityID,
		ExternalSystem: "jira",
		ExternalID:     externalID,
		ExternalKey:    externalKey,
		ExternalURL:    strings.TrimRight(cfg.BaseURL, "/") + "/browse/" + externalKey,
		SyncDirection:  intmodel.SyncDirectionBidirectional,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if _, err := s.links.Create(ctx, link); err != nil {
		return nil, code, body, err
	}
	if entityType == "alert" {
		_ = s.clario.AddAlertComment(ctx, bearerToken, entityID, "Linked Jira issue "+externalKey, map[string]any{
			"external_system": "jira",
			"external_key":    externalKey,
			"external_url":    link.ExternalURL,
		})
	}
	return link, code, body, nil
}

func (s *Service) SyncWebhookStatus(ctx context.Context, integration *intmodel.Integration, cfg intmodel.JiraConfig, issueID, externalStatus string) (*intmodel.ExternalTicketLink, error) {
	link, err := s.links.GetByExternal(ctx, "jira", issueID)
	if err != nil {
		return nil, err
	}
	mappedStatus := MapJiraStatusToClario(cfg, externalStatus)
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
	note := fmt.Sprintf("Status updated from Jira (%s): %s", link.ExternalKey, externalStatus)
	if err := s.clario.UpdateAlertStatus(ctx, systemToken, link.EntityID, mappedStatus, &note, nil); err != nil {
		errText := err.Error()
		link.SyncError = &errText
		_ = s.links.UpdateSync(ctx, link)
		return nil, err
	}
	_ = s.clario.AddAlertComment(ctx, systemToken, link.EntityID, note, map[string]any{
		"external_system": "jira",
		"external_key":    link.ExternalKey,
		"external_status": externalStatus,
	})
	link.SyncError = nil
	return link, s.links.UpdateSync(ctx, link)
}

func (s *Service) SyncLink(ctx context.Context, integration *intmodel.Integration, cfg intmodel.JiraConfig, link *intmodel.ExternalTicketLink) error {
	issue, _, _, err := s.client.GetIssue(ctx, cfg, link.ExternalID)
	if err != nil {
		return err
	}
	fields, _ := issue["fields"].(map[string]any)
	statusSection, _ := fields["status"].(map[string]any)
	status := stringValue(statusSection["name"])
	_, err = s.SyncWebhookStatus(ctx, integration, cfg, link.ExternalID, status)
	return err
}

func (s *Service) entityViewURL(entityType, entityID string) string {
	switch entityType {
	case "alert":
		return s.appURL + "/cyber/alerts/" + entityID
	case "action_item":
		return s.appURL + "/acta/action-items/" + entityID
	case "contract":
		return s.appURL + "/lex/contracts/" + entityID
	default:
		return s.appURL
	}
}

func extractSummary(entity map[string]any) string {
	if explanation, ok := entity["explanation"].(map[string]any); ok {
		return stringValue(explanation["summary"])
	}
	return ""
}
