package servicenow

import (
	"crypto/hmac"
	"fmt"
	"net/http"
	"strings"
)

func VerifyServiceNowToken(r *http.Request, sharedSecret string) error {
	token := strings.TrimSpace(r.Header.Get("X-ServiceNow-Token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Clario-Webhook-Secret"))
	}
	if token == "" {
		return fmt.Errorf("missing servicenow webhook token")
	}
	if !hmac.Equal([]byte(token), []byte(sharedSecret)) {
		return fmt.Errorf("invalid servicenow webhook token")
	}
	return nil
}
