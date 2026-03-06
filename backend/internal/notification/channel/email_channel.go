package channel

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"

	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/model"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// EmailConfig holds email channel configuration.
type EmailConfig struct {
	Provider   string // "smtp" or "sendgrid"
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
	TLSEnabled bool

	SendGridAPIKey string
	SendGridFrom   string
}

// EmailChannel sends email notifications via SMTP or SendGrid.
type EmailChannel struct {
	cfg       EmailConfig
	tmplSvc   EmailTemplateRenderer
	cb        *gobreaker.CircuitBreaker
	logger    zerolog.Logger
}

// NewEmailChannel creates a new EmailChannel.
func NewEmailChannel(cfg EmailConfig, tmplSvc EmailTemplateRenderer, logger zerolog.Logger) *EmailChannel {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "email_channel",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	return &EmailChannel{
		cfg:     cfg,
		tmplSvc: tmplSvc,
		cb:      cb,
		logger:  logger.With().Str("channel", "email").Logger(),
	}
}

// Name returns the channel name.
func (c *EmailChannel) Name() string { return model.ChannelEmail }

// Send delivers an email notification.
func (c *EmailChannel) Send(ctx context.Context, notif *model.Notification) *ChannelResult {
	// Extract email from notification data.
	email := c.extractEmail(notif)
	if email == "" {
		return &ChannelResult{
			Success:  false,
			Error:    fmt.Errorf("no email address available for user %s", notif.UserID),
			Metadata: map[string]interface{}{"reason": "no_email"},
		}
	}

	if !emailRegex.MatchString(email) {
		return &ChannelResult{
			Success:  false,
			Error:    fmt.Errorf("invalid email address: %s", email),
			Metadata: map[string]interface{}{"reason": "invalid_email"},
		}
	}

	// Render email template.
	subject, body, err := c.tmplSvc.RenderEmail(notif)
	if err != nil {
		c.logger.Warn().Err(err).Msg("template rendering failed, using fallback")
		subject = notif.Title
		body = notif.Body
	}

	// Send via configured provider with circuit breaker.
	_, cbErr := c.cb.Execute(func() (interface{}, error) {
		switch c.cfg.Provider {
		case "sendgrid":
			return nil, c.sendViaSendGrid(ctx, email, subject, body)
		default:
			return nil, c.sendViaSMTP(email, subject, body)
		}
	})

	if cbErr != nil {
		metrics.EmailSent.WithLabelValues(c.cfg.Provider, "failed").Inc()
		return &ChannelResult{
			Success:  false,
			Error:    cbErr,
			Metadata: map[string]interface{}{"provider": c.cfg.Provider},
		}
	}

	metrics.EmailSent.WithLabelValues(c.cfg.Provider, "sent").Inc()
	return &ChannelResult{
		Success:  true,
		Metadata: map[string]interface{}{"provider": c.cfg.Provider},
	}
}

func (c *EmailChannel) sendViaSMTP(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", c.cfg.SMTPHost, c.cfg.SMTPPort)
	from := c.cfg.SMTPFrom

	// Build MIME message.
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	if c.cfg.TLSEnabled {
		return c.sendSMTPWithTLS(addr, to, from, msg.Bytes())
	}

	var auth smtp.Auth
	if c.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", c.cfg.SMTPUser, c.cfg.SMTPPass, c.cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, from, []string{to}, msg.Bytes())
}

func (c *EmailChannel) sendSMTPWithTLS(addr, to, from string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if c.cfg.SMTPUser != "" {
		auth := smtp.PlainAuth("", c.cfg.SMTPUser, c.cfg.SMTPPass, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

func (c *EmailChannel) sendViaSendGrid(ctx context.Context, to, subject, htmlBody string) error {
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{"to": []map[string]string{{"email": to}}},
		},
		"from":    map[string]string{"email": c.cfg.SendGridFrom},
		"subject": subject,
		"content": []map[string]string{
			{"type": "text/html", "value": htmlBody},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendgrid payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sendgrid request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.SendGridAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *EmailChannel) extractEmail(notif *model.Notification) string {
	if len(notif.Data) == 0 {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal(notif.Data, &data); err != nil {
		return ""
	}
	if email, ok := data["email"].(string); ok {
		return email
	}
	if email, ok := data["user_email"].(string); ok {
		return email
	}
	return ""
}

// ExtractFromAddress extracts the email portion from a "Name <email>" format.
func ExtractFromAddress(from string) string {
	if idx := strings.Index(from, "<"); idx >= 0 {
		end := strings.Index(from, ">")
		if end > idx {
			return from[idx+1 : end]
		}
	}
	return from
}
