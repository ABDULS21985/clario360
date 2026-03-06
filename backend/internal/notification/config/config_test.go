package config

import "testing"

func TestValidate_SMTPValid(t *testing.T) {
	cfg := &Config{
		HTTPPort:          8089,
		EmailProvider:     "smtp",
		SMTPHost:          "smtp.example.com",
		SMTPPort:          587,
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
		DigestDailyUTCHour: 8,
		DigestWeeklyDay:    1,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidate_SMTPMissingHost(t *testing.T) {
	cfg := &Config{
		HTTPPort:      8089,
		EmailProvider: "smtp",
		SMTPHost:      "",
		SMTPPort:      587,
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing SMTP host")
	}
}

func TestValidate_SendGridMissingKey(t *testing.T) {
	cfg := &Config{
		HTTPPort:      8089,
		EmailProvider: "sendgrid",
		SendGridAPIKey: "",
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing SendGrid API key")
	}
}

func TestValidate_InvalidProvider(t *testing.T) {
	cfg := &Config{
		HTTPPort:      8089,
		EmailProvider: "mailgun",
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid email provider")
	}
}

func TestValidate_PingLessThanPong(t *testing.T) {
	cfg := &Config{
		HTTPPort:          8089,
		EmailProvider:     "smtp",
		SMTPHost:          "smtp.example.com",
		SMTPPort:          587,
		WSPingIntervalSec: 5,
		WSPongTimeoutSec:  10,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when ping interval <= pong timeout")
	}
}

func TestValidate_WebhookHMACTooShort(t *testing.T) {
	cfg := &Config{
		HTTPPort:          8089,
		EmailProvider:     "smtp",
		SMTPHost:          "smtp.example.com",
		SMTPPort:          587,
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
		WebhookHMACSecret: "short",
		DigestDailyUTCHour: 8,
		DigestWeeklyDay:    1,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for short HMAC secret")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := &Config{
		HTTPPort:      0,
		EmailProvider: "smtp",
		SMTPHost:      "smtp.example.com",
		SMTPPort:      587,
		WSPingIntervalSec: 30,
		WSPongTimeoutSec:  10,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid port")
	}
}
