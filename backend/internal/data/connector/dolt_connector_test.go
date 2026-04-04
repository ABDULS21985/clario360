package connector

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestDoltConnectorDefaultsBranch(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"host":     "localhost",
		"port":     3306,
		"database": "app",
		"username": "root",
		"password": "secret",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	connector, err := NewDoltConnector(raw, FactoryOptions{
		Limits: ConnectorLimits{StatementTimeout: 30 * time.Second},
		Logger: zerolog.Nop(),
	})
	if err != nil {
		t.Fatalf("NewDoltConnector() error = %v", err)
	}
	value, ok := connector.(*DoltConnector)
	if !ok {
		t.Fatalf("connector type = %T, want *DoltConnector", connector)
	}
	if value.config.Branch != "main" {
		t.Fatalf("Branch = %q, want main", value.config.Branch)
	}
}
