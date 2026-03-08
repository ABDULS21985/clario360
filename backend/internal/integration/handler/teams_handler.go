package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/integration/bot"
	botformatters "github.com/clario360/platform/internal/integration/bot/formatters"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intsvc "github.com/clario360/platform/internal/integration/service"
	teamssvc "github.com/clario360/platform/internal/integration/service/teams"
)

type TeamsHandler struct {
	service   *intsvc.IntegrationService
	api       *intsvc.ClarioAPIClient
	client    *teamssvc.Client
	botRouter *bot.Router
	producer  *events.Producer
	logger    zerolog.Logger
}

func NewTeamsHandler(
	service *intsvc.IntegrationService,
	api *intsvc.ClarioAPIClient,
	client *teamssvc.Client,
	botRouter *bot.Router,
	producer *events.Producer,
	logger zerolog.Logger,
) *TeamsHandler {
	return &TeamsHandler{
		service:   service,
		api:       api,
		client:    client,
		botRouter: botRouter,
		producer:  producer,
		logger:    logger.With().Str("component", "integration_teams_handler").Logger(),
	}
}

func (h *TeamsHandler) Messages(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyAndRestore(r, 1<<20)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "failed to read request body")
		return
	}
	var activity map[string]any
	if err := json.Unmarshal(body, &activity); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid teams activity payload")
		return
	}

	recipientID := nestedString(activity, "recipient", "id")
	serviceURL := firstNonEmpty(nestedString(activity, "serviceUrl"), nestedString(activity, "serviceURL"))
	conversationID := nestedString(activity, "conversation", "id")

	integration, cfg, err := h.findIntegration(r.Context(), recipientID, serviceURL, conversationID)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "no active teams integration matched this activity")
		return
	}

	claims, err := teamssvc.ValidateTeamsToken(r, cfg.BotAppID)
	if err != nil {
		h.logger.Warn().Err(err).Str("integration_id", integration.ID).Msg("rejected teams request with invalid token")
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "invalid teams bearer token")
		return
	}

	if strings.ToLower(strings.TrimSpace(firstNonEmpty(nestedString(activity, "type"), "message"))) != "message" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	text := strings.TrimSpace(nestedString(activity, "text"))
	text = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "/clario"), "clario"))
	subcommand, args := parseBotCommand(text)

	user, token := h.mapTeamsUserAndToken(r.Context(), integration.TenantID, activity)
	resp, routeErr := h.botRouter.Route(r.Context(), bottypes.BotCommand{
		Subcommand: subcommand,
		Args:       args,
		User:       user,
		Token:      token,
		TenantID:   integration.TenantID,
		Platform:   "teams",
		RawText:    text,
	})
	if routeErr != nil {
		resp = &bottypes.BotResponse{Text: "⚠️ " + routeErr.Error(), Ephemeral: true}
	}

	if _, _, err := h.client.SendActivityToConversation(r.Context(), cfg, claims.ServiceURL, conversationID, botformatters.TeamsResponse(resp)); err != nil {
		writeError(w, r, http.StatusBadGateway, "TEAMS_SEND_FAILED", err.Error())
		return
	}

	h.publishAudit(r.Context(), integration.TenantID, subcommand, nestedString(activity, "from", "id"), user, routeErr == nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *TeamsHandler) findIntegration(ctx context.Context, recipientID, serviceURL, conversationID string) (*intmodel.Integration, intmodel.TeamsConfig, error) {
	integration, configMap, err := h.service.FindActiveByType(ctx, intmodel.IntegrationTypeTeams, func(_ *intmodel.Integration, config map[string]any) bool {
		botAppID := strings.TrimSpace(stringValue(config["bot_app_id"]))
		cfgServiceURL := strings.TrimRight(strings.TrimSpace(stringValue(config["service_url"])), "/")
		cfgConversationID := strings.TrimSpace(stringValue(config["conversation_id"]))
		return (recipientID != "" && strings.EqualFold(botAppID, recipientID)) ||
			(serviceURL != "" && cfgServiceURL == strings.TrimRight(serviceURL, "/") && cfgConversationID == conversationID)
	})
	if err != nil {
		return nil, intmodel.TeamsConfig{}, err
	}
	var cfg intmodel.TeamsConfig
	if err := intsvc.DecodeInto(configMap, &cfg); err != nil {
		return nil, intmodel.TeamsConfig{}, err
	}
	return integration, cfg, nil
}

func (h *TeamsHandler) mapTeamsUserAndToken(ctx context.Context, tenantID string, activity map[string]any) (any, string) {
	user, err := teamssvc.MapTeamsUser(ctx, tenantID, activity, h.api.LookupUserByEmail)
	if err != nil {
		h.logger.Debug().Err(err).Str("tenant_id", tenantID).Msg("teams user is not linked")
		return nil, ""
	}
	token, err := h.api.MintUserToken(user)
	if err != nil {
		h.logger.Warn().Err(err).Str("user_id", user.ID).Msg("failed to mint teams user token")
		return user, ""
	}
	return user, token
}

func (h *TeamsHandler) publishAudit(ctx context.Context, tenantID, command, externalUserID string, mappedUser any, success bool) {
	actor := auditActorFromMappedUser(mappedUser)
	publishAuditEvent(ctx, h.producer, tenantID, "integration.teams.command.executed", actor, map[string]any{
		"platform":         "teams",
		"command":          command,
		"external_user_id": externalUserID,
		"mapped_user_id":   mappedUserID(mappedUser),
		"success":          success,
	})
}

