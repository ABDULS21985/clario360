package dto

import (
	"fmt"

	"github.com/clario360/platform/internal/notification/model"
)

// PreferenceUpdateRequest is the request body for updating notification preferences.
type PreferenceUpdateRequest struct {
	GlobalPrefs  *model.ChannelPreference                            `json:"global_prefs,omitempty"`
	PerTypePrefs map[model.NotificationType]model.ChannelPreference `json:"per_type_prefs,omitempty"`
	QuietHours   *model.QuietHours                                   `json:"quiet_hours,omitempty"`
	DigestConfig *model.DigestConfig                                 `json:"digest_config,omitempty"`
}

// Validate checks that the preference update request is well-formed.
func (r *PreferenceUpdateRequest) Validate() error {
	if r.QuietHours != nil && r.QuietHours.Enabled {
		if r.QuietHours.StartTime == "" || r.QuietHours.EndTime == "" {
			return fmt.Errorf("quiet_hours start_time and end_time are required when enabled")
		}
		if r.QuietHours.Timezone == "" {
			return fmt.Errorf("quiet_hours timezone is required when enabled")
		}
	}
	return nil
}

// WebhookCreateRequest is the request body for registering a new webhook.
type WebhookCreateRequest struct {
	Name        string                    `json:"name"`
	URL         string                    `json:"url"`
	Secret      string                    `json:"secret,omitempty"`
	Events      []string                  `json:"events"`
	Headers     map[string]string         `json:"headers,omitempty"`
	RetryPolicy *model.WebhookRetryPolicy `json:"retry_policy,omitempty"`
}

// Validate checks webhook creation request.
func (r *WebhookCreateRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	if len(r.Events) == 0 {
		return fmt.Errorf("at least one event is required")
	}
	return nil
}

// WebhookUpdateRequest is the request body for updating a webhook.
type WebhookUpdateRequest struct {
	Name        *string                   `json:"name,omitempty"`
	URL         *string                   `json:"url,omitempty"`
	Secret      *string                   `json:"secret,omitempty"`
	Events      []string                  `json:"events,omitempty"`
	Active      *bool                     `json:"active,omitempty"`
	Headers     map[string]string         `json:"headers,omitempty"`
	RetryPolicy *model.WebhookRetryPolicy `json:"retry_policy,omitempty"`
}

// TestNotificationRequest is the request body for sending a test notification.
type TestNotificationRequest struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	WebhookID string `json:"webhook_id,omitempty"`
}

// RetryFailedRequest is the request body for retrying failed deliveries.
type RetryFailedRequest struct {
	Channel         string   `json:"channel,omitempty"`
	Since           string   `json:"since,omitempty"`
	NotificationIDs []string `json:"notification_ids,omitempty"`
}
