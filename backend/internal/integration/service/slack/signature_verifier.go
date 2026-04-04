package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func VerifySlackSignature(r *http.Request, signingSecret string) error {
	signature := strings.TrimSpace(r.Header.Get("X-Slack-Signature"))
	if signature == "" {
		return fmt.Errorf("missing slack signature")
	}

	timestamp := strings.TrimSpace(r.Header.Get("X-Slack-Request-Timestamp"))
	if timestamp == "" {
		return fmt.Errorf("missing slack request timestamp")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid slack timestamp")
	}
	now := time.Now().UTC().Unix()
	if now-ts > 300 || ts-now > 300 {
		return fmt.Errorf("slack request timestamp expired")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read slack body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	return VerifySlackSignatureValues(signature, timestamp, body, signingSecret)
}

func VerifySlackSignatureValues(signature, timestamp string, body []byte, signingSecret string) error {
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte("v0:" + timestamp + ":" + string(body)))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid slack signature")
	}
	return nil
}
