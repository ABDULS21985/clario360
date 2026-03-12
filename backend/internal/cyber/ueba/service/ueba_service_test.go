package service

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/engine"
	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestConfigRoundTripPreservesUEBASettings(t *testing.T) {
	cfg := engine.DefaultConfig()
	cfg.BatchSize = 750
	cfg.CycleInterval = 10 * time.Minute
	cfg.MinMaturityForAlert = model.ProfileMaturityMature

	decoded, err := configFromDTO(configToDTO(cfg))
	if err != nil {
		t.Fatalf("configFromDTO returned error: %v", err)
	}

	if !reflect.DeepEqual(decoded, cfg) {
		t.Fatalf("decoded config = %#v, want %#v", decoded, cfg)
	}
}

func TestConfigFromDTORejectsInvalidDurations(t *testing.T) {
	req := configToDTO(engine.DefaultConfig())
	req.CycleInterval = "tomorrow"

	_, err := configFromDTO(req)
	if err == nil {
		t.Fatal("configFromDTO expected an error for invalid cycle interval")
	}
	if !strings.Contains(err.Error(), "invalid cycle_interval") {
		t.Fatalf("configFromDTO error = %v, want invalid cycle_interval", err)
	}
}

func TestMapProfilesSkipsNilEntriesAndFallsBackToEntityID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	items := []*model.UEBAProfile{
		nil,
		{
			ID:              uuid.New(),
			EntityID:        "svc-payments",
			EntityName:      "",
			EntityType:      model.EntityTypeServiceAccount,
			RiskScore:       87.5,
			RiskLevel:       model.RiskLevelHigh,
			AlertCount7D:    4,
			AlertCount30D:   9,
			ProfileMaturity: model.ProfileMaturityMature,
			LastSeenAt:      now,
			Status:          model.ProfileStatusActive,
		},
	}

	got := mapProfiles(items)
	if len(got) != 1 {
		t.Fatalf("mapProfiles length = %d, want 1", len(got))
	}

	if got[0].EntityName != "svc-payments" {
		t.Fatalf("EntityName = %q, want entity ID fallback", got[0].EntityName)
	}
	if got[0].EntityType != string(model.EntityTypeServiceAccount) {
		t.Fatalf("EntityType = %q, want %q", got[0].EntityType, model.EntityTypeServiceAccount)
	}
	if got[0].RiskLevel != string(model.RiskLevelHigh) {
		t.Fatalf("RiskLevel = %q, want %q", got[0].RiskLevel, model.RiskLevelHigh)
	}
}

func TestUpsertLRUStringDeduplicatesAndCapsEntries(t *testing.T) {
	got := upsertLRUString([]string{"10.0.0.2", "10.0.0.1"}, "10.0.0.2", 2)

	want := []string{"10.0.0.2", "10.0.0.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertLRUString duplicate = %#v, want %#v", got, want)
	}

	got = upsertLRUString([]string{"10.0.0.2", "10.0.0.1"}, "10.0.0.3", 2)
	want = []string{"10.0.0.3", "10.0.0.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertLRUString capped = %#v, want %#v", got, want)
	}
}

func TestUpsertTableAndTrimTablesKeepMostRelevantEntries(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Second)
	existing := []model.FrequencyEntry{
		{Name: "public.orders", Frequency: 0.9, LastAccessed: base.Add(-2 * time.Hour)},
		{Name: "public.users", Frequency: 0.3, LastAccessed: base.Add(-4 * time.Hour)},
	}

	updated := upsertTable(existing, "PUBLIC.USERS", base)
	if len(updated) != 2 {
		t.Fatalf("upsertTable length = %d, want 2", len(updated))
	}

	var users model.FrequencyEntry
	foundUsers := false
	for _, item := range updated {
		if strings.EqualFold(item.Name, "public.users") {
			users = item
			foundUsers = true
		}
	}
	if !foundUsers {
		t.Fatal("updated table list is missing public.users")
	}
	if users.LastAccessed != base {
		t.Fatalf("users.LastAccessed = %v, want %v", users.LastAccessed, base)
	}
	if users.Frequency != 0.3 {
		t.Fatalf("users.Frequency = %v, want 0.3", users.Frequency)
	}

	var oversized []model.FrequencyEntry
	for i := 0; i < 25; i++ {
		oversized = append(oversized, model.FrequencyEntry{
			Name:         string(rune('a' + i)),
			Frequency:    float64(25 - i),
			LastAccessed: base.Add(time.Duration(i) * time.Minute),
		})
	}

	trimmed := trimTables(oversized)
	if len(trimmed) != 20 {
		t.Fatalf("trimTables length = %d, want 20", len(trimmed))
	}
	if trimmed[0].Frequency < trimmed[len(trimmed)-1].Frequency {
		t.Fatalf("trimTables not sorted descending by frequency: first=%v last=%v", trimmed[0].Frequency, trimmed[len(trimmed)-1].Frequency)
	}
}
