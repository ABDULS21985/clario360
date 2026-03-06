package channel

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
)

// WebhookChannel delivers notifications to external HTTP endpoints.
type WebhookChannel struct {
	webhookRepo *repository.WebhookRepository
	client      *http.Client
	hmacSecret  string
	environment string
	logger      zerolog.Logger
}

// NewWebhookChannel creates a new WebhookChannel.
func NewWebhookChannel(
	webhookRepo *repository.WebhookRepository,
	timeout time.Duration,
	hmacSecret string,
	environment string,
	logger zerolog.Logger,
) *WebhookChannel {
	client := &http.Client{
		Timeout: timeout,
		// Never follow redirects to prevent SSRF via redirect to internal services.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &WebhookChannel{
		webhookRepo: webhookRepo,
		client:      client,
		hmacSecret:  hmacSecret,
		environment: environment,
		logger:      logger.With().Str("channel", "webhook").Logger(),
	}
}

// Name returns the channel name.
func (c *WebhookChannel) Name() string { return model.ChannelWebhook }

// Send delivers the notification to all matching webhooks for the tenant.
func (c *WebhookChannel) Send(ctx context.Context, notif *model.Notification) *ChannelResult {
	webhooks, err := c.webhookRepo.GetActiveForEvent(ctx, notif.TenantID, string(notif.Type))
	if err != nil {
		return &ChannelResult{Success: false, Error: fmt.Errorf("load webhooks: %w", err)}
	}

	if len(webhooks) == 0 {
		return &ChannelResult{
			Success:  true,
			Metadata: map[string]interface{}{"webhooks_matched": 0},
		}
	}

	var errs []string
	delivered := 0

	for _, wh := range webhooks {
		if err := c.deliverToWebhook(ctx, &wh, notif); err != nil {
			errs = append(errs, fmt.Sprintf("webhook %s: %v", wh.ID, err))
			metrics.WebhookDeliveries.WithLabelValues("failed").Inc()
		} else {
			delivered++
			metrics.WebhookDeliveries.WithLabelValues("delivered").Inc()
		}
	}

	if len(errs) > 0 && delivered == 0 {
		return &ChannelResult{
			Success:  false,
			Error:    fmt.Errorf("all webhooks failed: %s", strings.Join(errs, "; ")),
			Metadata: map[string]interface{}{"webhooks_matched": len(webhooks), "delivered": delivered},
		}
	}

	return &ChannelResult{
		Success:  true,
		Metadata: map[string]interface{}{"webhooks_matched": len(webhooks), "delivered": delivered},
	}
}

func (c *WebhookChannel) deliverToWebhook(ctx context.Context, wh *model.Webhook, notif *model.Notification) error {
	// Validate webhook URL.
	if err := c.validateURL(wh.URL); err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Build payload.
	payload := map[string]interface{}{
		"event":      string(notif.Type),
		"data":       notif,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"webhook_id": wh.ID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Clario360-Event", string(notif.Type))
	req.Header.Set("User-Agent", "Clario360-Webhook/1.0")

	// HMAC signing.
	signingSecret := c.hmacSecret
	if wh.Secret != nil && *wh.Secret != "" {
		signingSecret = *wh.Secret
	}
	if signingSecret != "" {
		mac := hmac.New(sha256.New, []byte(signingSecret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Clario360-Signature", "sha256="+sig)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf("webhook returned %d (permanent)", resp.StatusCode)
	}

	return fmt.Errorf("webhook returned %d (retriable)", resp.StatusCode)
}

func (c *WebhookChannel) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	// Must be HTTPS in production.
	if c.environment != "development" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must be HTTPS in production")
	}

	// Basic SSRF protection: reject private/loopback IPs.
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL must not point to private IP")
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	// IPv6 loopback.
	if ip.Equal(net.IPv6loopback) {
		return true
	}
	// IPv6 unique local (fc00::/7).
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}
