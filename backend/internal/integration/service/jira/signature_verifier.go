package jira

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func VerifyJiraSignature(r *http.Request, sharedSecret string) error {
	signature := strings.TrimSpace(r.Header.Get("X-Hub-Signature"))
	if signature == "" {
		signature = strings.TrimSpace(r.Header.Get("X-Atlassian-Webhook-Signature"))
	}
	if signature == "" {
		return fmt.Errorf("missing jira signature")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read jira body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	return VerifyJiraSignatureValues(signature, body, sharedSecret)
}

func VerifyJiraSignatureValues(signature string, body []byte, sharedSecret string) error {
	mac := hmac.New(sha256.New, []byte(sharedSecret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid jira signature")
	}
	return nil
}
