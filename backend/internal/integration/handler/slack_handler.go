package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	iamdto "github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/integration/bot"
	botformatters "github.com/clario360/platform/internal/integration/bot/formatters"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intsvc "github.com/clario360/platform/internal/integration/service"
	slacksvc "github.com/clario360/platform/internal/integration/service/slack"
)

var slackMentionPattern = regexp.MustCompile(`^<@([A-Z0-9]+)(?:\|[^>]+)?>$`)

type slackOAuthState struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Name     string `json:"name,omitempty"`
}

type slackCommandPayload struct {
	Command     string
	Text        string
	TeamID      string
	ChannelID   string
	UserID      string
	ResponseURL string
}

type slackInteractionPayload struct {
	Type      string `json:"type"`
	ResponseURL string `json:"response_url"`
	User      struct {
		ID string `json:"id"`
	} `json:"user"`
	Team struct {
		ID string `json:"id"`
	} `json:"team"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
	Container struct {
		MessageTS string `json:"message_ts"`
		ChannelID string `json:"channel_id"`
	} `json:"container"`
	Message struct {
		TS string `json:"ts"`
	} `json:"message"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
}

type SlackHandler struct {
	service         *intsvc.IntegrationService
	api             *intsvc.ClarioAPIClient
	client          *slacksvc.Client
	userMapper      *slacksvc.UserMapper
	botRouter       *bot.Router
	producer        *events.Producer
	redis           *redis.Client
	oauthCfg        slacksvc.OAuthConfig
	signingSecret   string
	publicAppURL    string
	stateTTL        time.Duration
	logger          zerolog.Logger
}

func NewSlackHandler(
	service *intsvc.IntegrationService,
	api *intsvc.ClarioAPIClient,
	client *slacksvc.Client,
	userMapper *slacksvc.UserMapper,
	botRouter *bot.Router,
	producer *events.Producer,
	redis *redis.Client,
	oauthCfg slacksvc.OAuthConfig,
	signingSecret string,
	publicAppURL string,
	stateTTL time.Duration,
	logger zerolog.Logger,
) *SlackHandler {
	if stateTTL <= 0 {
		stateTTL = 15 * time.Minute
	}
	return &SlackHandler{
		service:       service,
		api:           api,
		client:        client,
		userMapper:    userMapper,
		botRouter:     botRouter,
		producer:      producer,
		redis:         redis,
		oauthCfg:      oauthCfg,
		signingSecret: signingSecret,
		publicAppURL:  strings.TrimRight(publicAppURL, "/"),
		stateTTL:      stateTTL,
		logger:        logger.With().Str("component", "integration_slack_handler").Logger(),
	}
}

func (h *SlackHandler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.redis == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STATE_STORE_UNAVAILABLE", "oauth state store is unavailable")
		return
	}
	stateID := events.GenerateUUID()
	state := slackOAuthState{
		TenantID: tenantID,
		UserID:   user.ID,
		Name:     strings.TrimSpace(r.URL.Query().Get("name")),
	}
	raw, err := json.Marshal(state)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "STATE_ENCODE_FAILED", "failed to initialize oauth state")
		return
	}
	if err := h.redis.Set(r.Context(), "integration:slack:oauth:"+stateID, raw, h.stateTTL).Err(); err != nil {
		writeError(w, r, http.StatusInternalServerError, "STATE_STORE_FAILED", "failed to persist oauth state")
		return
	}
	http.Redirect(w, r, slacksvc.BuildOAuthURL(h.oauthCfg, stateID), http.StatusFound)
}

func (h *SlackHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.redis == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STATE_STORE_UNAVAILABLE", "oauth state store is unavailable")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	stateID := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || stateID == "" {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_CALLBACK", "missing code or state")
		return
	}

	stateRaw, err := h.redis.GetDel(r.Context(), "integration:slack:oauth:"+stateID).Bytes()
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state is missing or expired")
		return
	}
	var state slackOAuthState
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state is invalid")
		return
	}

	payload, err := slacksvc.ExchangeCode(r.Context(), h.oauthCfg, code)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "SLACK_OAUTH_FAILED", err.Error())
		return
	}

	team, _ := payload["team"].(map[string]any)
	incomingWebhook, _ := payload["incoming_webhook"].(map[string]any)
	config := map[string]any{
		"bot_token":      stringValue(payload["access_token"]),
		"team_id":        firstNonEmpty(stringValue(team["id"]), stringValue(payload["team_id"])),
		"team_name":      firstNonEmpty(stringValue(team["name"]), stringValue(payload["team_name"])),
		"signing_secret": h.signingSecret,
	}
	if webhookURL := stringValue(incomingWebhook["url"]); webhookURL != "" {
		config["incoming_webhook_url"] = webhookURL
	}
	if channelID := stringValue(incomingWebhook["channel_id"]); channelID != "" {
		config["channel_id"] = channelID
	}

	name := firstNonEmpty(state.Name, "Slack "+firstNonEmpty(stringValue(team["name"]), stringValue(payload["team_name"]), "Workspace"))
	var integration *intmodel.Integration
	if config["channel_id"] != nil || config["incoming_webhook_url"] != nil {
		req := &intdto.CreateIntegrationRequest{
			Type:   intmodel.IntegrationTypeSlack,
			Name:   name,
			Config: config,
		}
		integration, err = h.service.Create(r.Context(), state.TenantID, state.UserID, req, &intsvc.AuditActor{UserID: state.UserID})
	} else {
		integration, err = h.service.CreateSetupPending(r.Context(), state.TenantID, state.UserID, intmodel.IntegrationTypeSlack, name, "Slack integration awaiting channel configuration", config, &intsvc.AuditActor{UserID: state.UserID})
	}
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "SLACK_INSTALL_FAILED", err.Error())
		return
	}

	redirectURL := h.publicAppURL + "/admin/integrations/" + integration.ID
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *SlackHandler) Events(w http.ResponseWriter, r *http.Request) {
	if !h.verifySignature(w, r) {
		return
	}

	var payload struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid slack payload")
		return
	}
	if payload.Type == "url_verification" {
		writeJSON(w, http.StatusOK, map[string]string{"challenge": payload.Challenge})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *SlackHandler) Commands(w http.ResponseWriter, r *http.Request) {
	if !h.verifySignature(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, slackErrorPayload("invalid slash command payload"))
		return
	}
	command := slackCommandPayload{
		Command:     r.PostForm.Get("command"),
		Text:        r.PostForm.Get("text"),
		TeamID:      r.PostForm.Get("team_id"),
		ChannelID:   r.PostForm.Get("channel_id"),
		UserID:      r.PostForm.Get("user_id"),
		ResponseURL: r.PostForm.Get("response_url"),
	}
	if command.TeamID == "" {
		writeJSON(w, http.StatusOK, slackErrorPayload("slack team_id is required"))
		return
	}

	type result struct {
		response *bottypes.BotResponse
		err      error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := h.executeSlashCommand(context.WithoutCancel(r.Context()), command)
		resCh <- result{response: resp, err: err}
	}()

	select {
	case res := <-resCh:
		writeJSON(w, http.StatusOK, h.slackResponsePayload(res.response, res.err))
	case <-time.After(2500 * time.Millisecond):
		writeJSON(w, http.StatusOK, map[string]any{
			"response_type": "ephemeral",
			"text":          "⏳ Processing your request...",
		})
		if command.ResponseURL != "" {
			go func() {
				res := <-resCh
				if err := h.client.PostResponseURL(context.Background(), command.ResponseURL, h.slackResponsePayload(res.response, res.err)); err != nil {
					h.logger.Warn().Err(err).Str("team_id", command.TeamID).Msg("failed to post deferred slack command response")
				}
			}()
		}
	}
}

func (h *SlackHandler) Interactions(w http.ResponseWriter, r *http.Request) {
	if !h.verifySignature(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, slackErrorPayload("invalid slack interaction payload"))
		return
	}
	rawPayload := r.PostForm.Get("payload")
	if rawPayload == "" {
		writeJSON(w, http.StatusOK, slackErrorPayload("missing interaction payload"))
		return
	}

	var payload slackInteractionPayload
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		writeJSON(w, http.StatusOK, slackErrorPayload("invalid interaction payload"))
		return
	}
	if payload.Team.ID == "" || len(payload.Actions) == 0 {
		writeJSON(w, http.StatusOK, slackErrorPayload("interaction payload is incomplete"))
		return
	}

	integration, cfg, err := h.findIntegration(r.Context(), payload.Team.ID)
	if err != nil {
		writeJSON(w, http.StatusOK, slackErrorPayload("no active Slack integration matched this workspace"))
		return
	}

	user, token := h.mappedSlackUserAndToken(r.Context(), integration, cfg, payload.User.ID)
	action := payload.Actions[0]
	cmd := bottypes.BotCommand{
		User:     user,
		Token:    token,
		TenantID: integration.TenantID,
		Platform: "slack",
		RawText:  action.ActionID,
	}
	switch action.ActionID {
	case "clario_ack":
		cmd.Subcommand = "ack"
		cmd.Args = []string{action.Value}
	case "clario_investigate":
		cmd.Subcommand = "investigate"
		cmd.Args = []string{action.Value}
	default:
		writeJSON(w, http.StatusOK, slackErrorPayload("unsupported interaction"))
		return
	}

	resp, err := h.botRouter.Route(r.Context(), cmd)
	if err != nil {
		writeJSON(w, http.StatusOK, slackErrorPayload(err.Error()))
		return
	}

	channelID := firstNonEmpty(payload.Channel.ID, payload.Container.ChannelID)
	messageTS := firstNonEmpty(payload.Message.TS, payload.Container.MessageTS)
	if action.ActionID == "clario_ack" && channelID != "" && messageTS != "" {
		update := botformatters.SlackResponse(resp)
		delete(update, "response_type")
		if _, _, err := h.client.UpdateMessage(r.Context(), cfg.BotToken, channelID, messageTS, update); err != nil {
			h.logger.Warn().Err(err).Str("integration_id", integration.ID).Msg("failed to update slack message after ack")
		}
	}
	if action.ActionID == "clario_investigate" && channelID != "" && messageTS != "" {
		reply := botformatters.SlackResponse(resp)
		delete(reply, "response_type")
		if _, _, err := h.client.PostThreadReply(r.Context(), cfg.BotToken, channelID, messageTS, reply); err != nil {
			h.logger.Warn().Err(err).Str("integration_id", integration.ID).Msg("failed to post slack thread reply")
		}
	}

	h.publishCommandAudit(r.Context(), integration.TenantID, action.ActionID, payload.User.ID, user, true)
	writeJSON(w, http.StatusOK, map[string]any{
		"response_type": "ephemeral",
		"text":          resp.Text,
	})
}

func (h *SlackHandler) verifySignature(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(h.signingSecret) == "" {
		writeError(w, r, http.StatusUnauthorized, "SIGNATURE_INVALID", "slack signing secret is not configured")
		return false
	}
	if err := slacksvc.VerifySlackSignature(r, h.signingSecret); err != nil {
		h.logger.Warn().Err(err).Msg("rejected slack request with invalid signature")
		writeError(w, r, http.StatusUnauthorized, "SIGNATURE_INVALID", "invalid slack signature")
		return false
	}
	return true
}

func (h *SlackHandler) findIntegration(ctx context.Context, teamID string) (*intmodel.Integration, intmodel.SlackConfig, error) {
	integration, configMap, err := h.service.FindActiveByType(ctx, intmodel.IntegrationTypeSlack, func(_ *intmodel.Integration, config map[string]any) bool {
		return strings.EqualFold(strings.TrimSpace(stringValue(config["team_id"])), strings.TrimSpace(teamID))
	})
	if err != nil {
		return nil, intmodel.SlackConfig{}, err
	}
	var cfg intmodel.SlackConfig
	if err := intsvc.DecodeInto(configMap, &cfg); err != nil {
		return nil, intmodel.SlackConfig{}, err
	}
	return integration, cfg, nil
}

func (h *SlackHandler) executeSlashCommand(ctx context.Context, command slackCommandPayload) (*bottypes.BotResponse, error) {
	integration, cfg, err := h.findIntegration(ctx, command.TeamID)
	if err != nil {
		return nil, err
	}
	subcommand, args := parseBotCommand(command.Text)
	if subcommand == "assign" && len(args) >= 2 {
		args[1] = h.resolveSlackTarget(ctx, integration, cfg, args[1])
	}

	user, token := h.mappedSlackUserAndToken(ctx, integration, cfg, command.UserID)
	resp, err := h.botRouter.Route(ctx, bottypes.BotCommand{
		Subcommand: subcommand,
		Args:       args,
		User:       user,
		Token:      token,
		TenantID:   integration.TenantID,
		Platform:   "slack",
		RawText:    command.Text,
	})
	if err != nil {
		h.publishCommandAudit(ctx, integration.TenantID, subcommand, command.UserID, user, false)
		return nil, err
	}
	h.publishCommandAudit(ctx, integration.TenantID, subcommand, command.UserID, user, true)
	return resp, nil
}

func (h *SlackHandler) mappedSlackUserAndToken(ctx context.Context, integration *intmodel.Integration, cfg intmodel.SlackConfig, slackUserID string) (user *iamdto.UserResponse, token string) {
	if slackUserID == "" {
		return nil, ""
	}
	mappedUser, err := h.userMapper.MapSlackUser(ctx, cfg.BotToken, integration.TenantID, cfg.TeamID, slackUserID, h.api.LookupUserByEmail)
	if err != nil {
		h.logger.Debug().Err(err).Str("slack_user_id", slackUserID).Str("tenant_id", integration.TenantID).Msg("slack user is not linked")
		return nil, ""
	}
	userToken, err := h.api.MintUserToken(mappedUser)
	if err != nil {
		h.logger.Warn().Err(err).Str("user_id", mappedUser.ID).Msg("failed to mint user token for slack command")
		return mappedUser, ""
	}
	return mappedUser, userToken
}

func (h *SlackHandler) resolveSlackTarget(ctx context.Context, integration *intmodel.Integration, cfg intmodel.SlackConfig, raw string) string {
	matches := slackMentionPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 2 {
		return raw
	}
	target, err := h.userMapper.MapSlackUser(ctx, cfg.BotToken, integration.TenantID, cfg.TeamID, matches[1], h.api.LookupUserByEmail)
	if err != nil {
		h.logger.Debug().Err(err).Str("slack_user_id", matches[1]).Msg("failed to resolve slack mention")
		return raw
	}
	return target.ID
}

func (h *SlackHandler) publishCommandAudit(ctx context.Context, tenantID, command, externalUserID string, mappedUser any, success bool) {
	actor := auditActorFromMappedUser(mappedUser)
	data := map[string]any{
		"platform":         "slack",
		"command":          command,
		"external_user_id": externalUserID,
		"mapped_user_id":   mappedUserID(mappedUser),
		"success":          success,
	}
	publishAuditEvent(ctx, h.producer, tenantID, "integration.slack.command.executed", actor, data)
}

func (h *SlackHandler) slackResponsePayload(resp *bottypes.BotResponse, err error) map[string]any {
	if err != nil {
		return slackErrorPayload(err.Error())
	}
	if resp == nil {
		return slackErrorPayload("command returned an empty response")
	}
	return botformatters.SlackResponse(resp)
}

func parseBotCommand(text string) (string, []string) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "/clario"))
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "help", nil
	}
	return strings.ToLower(fields[0]), fields[1:]
}

func slackErrorPayload(message string) map[string]any {
	return map[string]any{
		"response_type": "ephemeral",
		"text":          "⚠️ " + strings.TrimSpace(message),
	}
}
