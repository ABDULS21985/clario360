package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"
)

type webhookTestResult struct {
	Success        bool   `json:"success"`
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `json:"response_body"`
}

func deliverWebhookTest(url string, payload []byte, secret *string, headers map[string]string) webhookTestResult {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return webhookTestResult{Success: false, ResponseBody: "failed to create request: " + err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Clario360-Webhook/1.0")

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Sign payload with HMAC-SHA256 if secret is available
	if secret != nil && *secret != "" {
		mac := hmac.New(sha256.New, []byte(*secret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return webhookTestResult{Success: false, ResponseBody: "request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return webhookTestResult{
		Success:        resp.StatusCode >= 200 && resp.StatusCode < 300,
		ResponseStatus: resp.StatusCode,
		ResponseBody:   string(body),
	}
}
