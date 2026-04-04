//go:build integration

package connector

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

func integrationFactoryOptions() FactoryOptions {
	return FactoryOptions{
		Limits: ConnectorLimits{
			MaxPoolSize:      4,
			StatementTimeout: 30 * time.Second,
			ConnectTimeout:   10 * time.Second,
			MaxSampleRows:    10,
			MaxTables:        100,
			APIRateLimit:     5,
		},
		Logger: zerolog.Nop(),
	}
}

func mustRawJSON(t testing.TB, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	return raw
}

func integrationSourceContext() (uuid.UUID, uuid.UUID) {
	return uuid.New(), uuid.New()
}

func waitForCondition(t testing.TB, timeout time.Duration, fn func(context.Context) (bool, error)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		ok, err := fn(ctx)
		if err == nil && ok {
			return
		}
		select {
		case <-ctx.Done():
			if err != nil {
				t.Fatalf("condition failed before timeout: %v", err)
			}
			t.Fatalf("condition not satisfied within %s", timeout)
		case <-ticker.C:
		}
	}
}

func requireDiscoveredTable(t testing.TB, schema *model.DiscoveredSchema, tableName string) model.DiscoveredTable {
	t.Helper()
	for _, table := range schema.Tables {
		if strings.EqualFold(table.Name, tableName) {
			return table
		}
	}
	t.Fatalf("table %q not found in discovered schema: %+v", tableName, schema.Tables)
	return model.DiscoveredTable{}
}

func requireDiscoveredColumn(t testing.TB, table model.DiscoveredTable, columnName string) model.DiscoveredColumn {
	t.Helper()
	for _, column := range table.Columns {
		if strings.EqualFold(column.Name, columnName) {
			return column
		}
	}
	t.Fatalf("column %q not found in table %q: %+v", columnName, table.Name, table.Columns)
	return model.DiscoveredColumn{}
}
