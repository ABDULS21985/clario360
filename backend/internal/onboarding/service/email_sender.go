package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/channel"
	notificationmodel "github.com/clario360/platform/internal/notification/model"
)

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, email, orgName, adminName, otp string) error
	SendInvitationEmail(ctx context.Context, email, organizationName, inviterName, roleName, rawToken string, message *string, expiresAt time.Time) error
	SendWelcomeEmail(ctx context.Context, email, organizationName, firstName string) error
}

type ChannelEmailSender struct {
	appURL       string
	emailChannel *channel.EmailChannel
	logger       zerolog.Logger
}

func NewChannelEmailSender(appURL string, emailChannel *channel.EmailChannel, logger zerolog.Logger) *ChannelEmailSender {
	return &ChannelEmailSender{
		appURL:       strings.TrimRight(appURL, "/"),
		emailChannel: emailChannel,
		logger:       logger.With().Str("component", "onboarding_email_sender").Logger(),
	}
}

func (s *ChannelEmailSender) SendVerificationEmail(ctx context.Context, email, orgName, adminName, otp string) error {
	if s.emailChannel == nil {
		return nil
	}
	data, _ := json.Marshal(map[string]any{
		"email":    email,
		"org_name": orgName,
		"admin":    adminName,
	})
	result := s.emailChannel.Send(ctx, &notificationmodel.Notification{
		ID:       uuid.NewString(),
		TenantID: "",
		UserID:   uuid.NewString(),
		Type:     notificationmodel.NotificationType("generic"),
		Category: notificationmodel.CategorySystem,
		Priority: notificationmodel.PriorityHigh,
		Title:    "Verify your Clario 360 email",
		Body:     fmt.Sprintf("Hello %s, your verification code for %s is %s. It is valid for 10 minutes.", adminName, orgName, otp),
		Data:     data,
	})
	if !result.Success {
		return result.Error
	}
	return nil
}

func (s *ChannelEmailSender) SendInvitationEmail(ctx context.Context, email, organizationName, inviterName, roleName, rawToken string, message *string, expiresAt time.Time) error {
	if s.emailChannel == nil {
		return nil
	}
	actionURL := fmt.Sprintf("%s/invite?token=%s", s.appURL, rawToken)
	data, _ := json.Marshal(map[string]any{
		"email":      email,
		"action_url": actionURL,
		"org_name":   organizationName,
		"inviter":    inviterName,
		"role_name":  roleName,
		"message":    message,
	})
	body := fmt.Sprintf("%s has invited you to join %s on Clario 360 as %s. This invitation expires on %s.",
		inviterName,
		organizationName,
		roleName,
		expiresAt.UTC().Format("Jan 02, 2006 15:04 UTC"),
	)
	if message != nil && strings.TrimSpace(*message) != "" {
		body += " Personal note: " + strings.TrimSpace(*message)
	}
	result := s.emailChannel.Send(ctx, &notificationmodel.Notification{
		ID:        uuid.NewString(),
		TenantID:  "",
		UserID:    uuid.NewString(),
		Type:      notificationmodel.NotificationType("generic"),
		Category:  notificationmodel.CategorySystem,
		Priority:  notificationmodel.PriorityMedium,
		Title:     "You have been invited to Clario 360",
		Body:      body,
		Data:      data,
		ActionURL: actionURL,
	})
	if !result.Success {
		return result.Error
	}
	return nil
}

func (s *ChannelEmailSender) SendWelcomeEmail(ctx context.Context, email, organizationName, firstName string) error {
	if s.emailChannel == nil {
		return nil
	}
	data, _ := json.Marshal(map[string]any{
		"email":      email,
		"action_url": s.appURL + "/dashboard",
		"org_name":   organizationName,
	})
	result := s.emailChannel.Send(ctx, &notificationmodel.Notification{
		ID:        uuid.NewString(),
		TenantID:  "",
		UserID:    uuid.NewString(),
		Type:      notificationmodel.NotificationType("generic"),
		Category:  notificationmodel.CategorySystem,
		Priority:  notificationmodel.PriorityLow,
		Title:     "Welcome to Clario 360",
		Body:      fmt.Sprintf("Welcome %s. Your %s workspace is ready.", firstName, organizationName),
		Data:      data,
		ActionURL: s.appURL + "/dashboard",
	})
	if !result.Success {
		return result.Error
	}
	return nil
}
