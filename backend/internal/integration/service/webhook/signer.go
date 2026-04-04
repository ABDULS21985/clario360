package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

func SignWebhookRequest(body []byte, secret string, timestamp int64, requestID string) map[string]string {
	headers := map[string]string{
		"X-Clario-Timestamp":  strconv.FormatInt(timestamp, 10),
		"X-Clario-Request-ID": requestID,
	}
	if secret == "" {
		return headers
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	headers["X-Clario-Signature"] = hex.EncodeToString(mac.Sum(nil))
	return headers
}

func VerifySignedWebhook(body []byte, secret, timestamp, signature string) error {
	if secret == "" {
		return fmt.Errorf("shared secret is not configured")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
