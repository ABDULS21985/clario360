package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
)

// TemplateFuncs provides common functions available in all notification templates.
var TemplateFuncs = texttemplate.FuncMap{
	"title": strings.Title,
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"formatDate": func(v interface{}) string {
		switch t := v.(type) {
		case time.Time:
			return t.UTC().Format("Jan 02, 2006 at 15:04 UTC")
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return parsed.UTC().Format("Jan 02, 2006 at 15:04 UTC")
			}
			return t
		default:
			return fmt.Sprintf("%v", v)
		}
	},
	"truncate": func(n int, s string) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "..."
	},
}

// HTMLTemplateFuncs mirrors TemplateFuncs for html/template.
var HTMLTemplateFuncs = htmltemplate.FuncMap{
	"title": strings.Title,
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"formatDate": func(v interface{}) string {
		switch t := v.(type) {
		case time.Time:
			return t.UTC().Format("Jan 02, 2006 at 15:04 UTC")
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return parsed.UTC().Format("Jan 02, 2006 at 15:04 UTC")
			}
			return t
		default:
			return fmt.Sprintf("%v", v)
		}
	},
	"truncate": func(n int, s string) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "..."
	},
}

// TemplateService handles rendering of notification templates.
type TemplateService struct {
	baseLayout string
	templates  map[string]string // notifType → body template content
	logger     zerolog.Logger
}

// NewTemplateService creates a new TemplateService with embedded templates.
func NewTemplateService(logger zerolog.Logger) *TemplateService {
	svc := &TemplateService{
		templates: make(map[string]string),
		logger:    logger.With().Str("component", "template_service").Logger(),
	}

	svc.baseLayout = baseLayoutTemplate
	svc.templates["alert.created"] = alertNotificationTemplate
	svc.templates["alert.escalated"] = alertEscalatedTemplate
	svc.templates["task.assigned"] = taskAssignedTemplate
	svc.templates["security.incident"] = securityIncidentTemplate
	svc.templates["system.maintenance"] = systemMaintenanceTemplate
	svc.templates["pipeline.failed"] = pipelineFailedTemplate
	svc.templates["contract.expiring"] = contractExpiringTemplate
	svc.templates["generic"] = genericTemplate
	svc.templates["digest"] = digestTemplate

	return svc
}

// RenderEmail renders the subject and HTML body for an email notification.
func (s *TemplateService) RenderEmail(notif *model.Notification) (string, string, error) {
	// Render subject via text/template (plain text).
	subject, err := s.renderText(notif.Title, s.buildTemplateData(notif))
	if err != nil {
		subject = notif.Title
	}

	// Select template for notification type.
	bodyTmplStr, ok := s.templates[string(notif.Type)]
	if !ok {
		bodyTmplStr = s.templates["generic"]
	}

	// Render body via html/template (XSS-safe).
	body, err := s.renderHTML(bodyTmplStr, s.buildTemplateData(notif))
	if err != nil {
		return subject, notif.Body, nil
	}

	// Wrap in base layout.
	layoutData := map[string]interface{}{
		"Content":      htmltemplate.HTML(body),
		"PrimaryColor": "#1B5E20",
		"AccentColor":  "#C6A962",
		"Year":         time.Now().Year(),
	}
	wrappedBody, err := s.renderHTML(s.baseLayout, layoutData)
	if err != nil {
		return subject, body, nil
	}

	return subject, wrappedBody, nil
}

// RenderText renders a template string as plain text.
func (s *TemplateService) RenderText(tmplStr string, data map[string]interface{}) (string, error) {
	return s.renderText(tmplStr, data)
}

func (s *TemplateService) renderText(tmplStr string, data interface{}) (string, error) {
	tmpl, err := texttemplate.New("t").Funcs(TemplateFuncs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse text template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute text template: %w", err)
	}
	return buf.String(), nil
}

func (s *TemplateService) renderHTML(tmplStr string, data interface{}) (string, error) {
	tmpl, err := htmltemplate.New("t").Funcs(HTMLTemplateFuncs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse html template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute html template: %w", err)
	}
	return buf.String(), nil
}

func (s *TemplateService) buildTemplateData(notif *model.Notification) map[string]interface{} {
	data := map[string]interface{}{
		"title":    notif.Title,
		"body":     notif.Body,
		"type":     string(notif.Type),
		"category": notif.Category,
		"priority": notif.Priority,
		"id":       notif.ID,
	}

	// Merge notification data into template data.
	if notif.Data != nil {
		var nd map[string]interface{}
		if err := json.Unmarshal(notif.Data, &nd); err == nil {
			for k, v := range nd {
				data[k] = v
			}
		}
	}

	return data
}

// Embedded templates.
const baseLayoutTemplate = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0; padding:0; font-family:Arial, sans-serif; background-color:#f5f5f5;">
<table width="100%" cellpadding="0" cellspacing="0" style="max-width:600px; margin:0 auto; background:#ffffff;">
<tr><td style="background-color:{{.PrimaryColor}}; padding:20px; text-align:center;">
<h1 style="color:#ffffff; margin:0; font-size:24px;">Clario 360</h1>
</td></tr>
<tr><td style="padding:30px;">{{.Content}}</td></tr>
<tr><td style="background-color:#f0f0f0; padding:15px; text-align:center; font-size:12px; color:#666;">
<p>&copy; {{.Year}} Clario 360. All rights reserved.</p>
</td></tr>
</table>
</body>
</html>`

const alertNotificationTemplate = `<h2 style="color:#d32f2f;">Security Alert</h2>
<p><strong>Priority:</strong> {{.priority}}</p>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#1B5E20; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Alert</a></p>{{end}}`

const alertEscalatedTemplate = `<h2 style="color:#d32f2f;">Alert Escalated</h2>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#d32f2f; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Alert</a></p>{{end}}`

const taskAssignedTemplate = `<h2 style="color:#1B5E20;">New Task Assigned</h2>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#1B5E20; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Task</a></p>{{end}}`

const securityIncidentTemplate = `<h2 style="color:#d32f2f;">SECURITY INCIDENT</h2>
<p style="background-color:#fce4ec; padding:15px; border-left:4px solid #d32f2f;">{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#d32f2f; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">Respond Now</a></p>{{end}}`

const systemMaintenanceTemplate = `<h2 style="color:#1565c0;">Scheduled Maintenance</h2>
<p>{{.body}}</p>`

const pipelineFailedTemplate = `<h2 style="color:#e65100;">Pipeline Failed</h2>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#e65100; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Pipeline</a></p>{{end}}`

const contractExpiringTemplate = `<h2 style="color:#C6A962;">Contract Expiring</h2>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#1B5E20; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Contract</a></p>{{end}}`

const genericTemplate = `<h2>{{.title}}</h2>
<p>{{.body}}</p>
{{if .action_url}}<p><a href="{{.action_url}}" style="background-color:#1B5E20; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View Details</a></p>{{end}}`

const digestTemplate = `<h2>Your Notification Summary</h2>
<p>You have <strong>{{.count}}</strong> unread notifications.</p>
{{range .items}}<div style="border-bottom:1px solid #eee; padding:10px 0;">
<strong>{{.Title}}</strong><br>
<span style="color:#666;">{{.Body}}</span>
</div>{{end}}
<p><a href="{{.dashboard_url}}" style="background-color:#1B5E20; color:#fff; padding:10px 20px; text-decoration:none; border-radius:4px;">View All Notifications</a></p>`
