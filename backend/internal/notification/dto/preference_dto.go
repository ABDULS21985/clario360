package dto

import (
	"fmt"

	"github.com/clario360/platform/internal/notification/model"
)

// PreferenceUpdateRequest is the request body for updating notification preferences.
type PreferenceUpdateRequest struct {
	GlobalPrefs  *model.ChannelPreference                   `json:"global_prefs,omitempty"`
	PerTypePrefs map[model.NotificationType]model.ChannelPreference `json:"per_type_prefs,omitempty"`
	QuietHours   *model.QuietHours                          `json:"quiet_hours,omitempty"`
	DigestConfig *model.DigestConfig                        `json:"digest_config,omitempty"`
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
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Secret     string   `json:"secret,omitempty"`
	EventTypes []string `json:"event_types"`
}

// Validate checks webhook creation request.
func (r *WebhookCreateRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

// WebhookUpdateRequest is the request body for updating a webhook.
type WebhookUpdateRequest struct {
	Name       *string  `json:"name,omitempty"`
	URL        *string  `json:"url,omitempty"`
	Secret     *string  `json:"secret,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
	Active     *bool    `json:"active,omitempty"`
}
