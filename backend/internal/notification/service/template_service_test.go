package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
)

func newTestTemplateService() *TemplateService {
	return NewTemplateService(zerolog.Nop())
}

func TestTemplateService_RenderEmail_AlertCreated(t *testing.T) {
	svc := newTestTemplateService()

	notif := &model.Notification{
		ID:       "notif-1",
		Type:     model.NotifAlertCreated,
		Category: model.CategorySecurity,
		Priority: model.PriorityCritical,
		Title:    "Critical Alert",
		Body:     "Ransomware detected on server-01",
		Data:     json.RawMessage(`{"action_url": "/cyber/alerts/123"}`),
	}

	subject, body, err := svc.RenderEmail(notif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if subject == "" {
		t.Error("expected non-empty subject")
	}
	if !strings.Contains(body, "Security Alert") {
		t.Error("expected body to contain 'Security Alert'")
	}
	if !strings.Contains(body, "Clario 360") {
		t.Error("expected body to contain layout branding")
	}
}

func TestTemplateService_RenderEmail_Generic(t *testing.T) {
	svc := newTestTemplateService()

	notif := &model.Notification{
		ID:       "notif-2",
		Type:     model.NotifPasswordExpiring,
		Category: model.CategorySystem,
		Priority: model.PriorityMedium,
		Title:    "Password Expiring",
		Body:     "Your password will expire in 7 days.",
		Data:     json.RawMessage(`{}`),
	}

	subject, body, err := svc.RenderEmail(notif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if subject != "Password Expiring" {
		t.Errorf("expected subject 'Password Expiring', got %q", subject)
	}
	if !strings.Contains(body, "Password Expiring") {
		t.Error("expected generic template to contain title")
	}
}

func TestTemplateService_RenderText(t *testing.T) {
	svc := newTestTemplateService()

	result, err := svc.RenderText("Hello {{.name}}, your task is {{.task}}", map[string]interface{}{
		"name": "Alice",
		"task": "Review Q4 Report",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Hello Alice, your task is Review Q4 Report" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestTemplateService_RenderText_MissingKey(t *testing.T) {
	svc := newTestTemplateService()

	// Missing keys in Go templates render as <no value> by default.
	result, err := svc.RenderText("Hello {{.name}}", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result even with missing key")
	}
}
