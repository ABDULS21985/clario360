package repository

import (
	"reflect"
	"testing"

	intmodel "github.com/clario360/platform/internal/integration/model"
)

func TestParseEventFiltersStructured(t *testing.T) {
	filters, err := parseEventFilters([]byte(`[{"event_types":["com.clario360.cyber.alert.created"],"severities":["critical"]}]`))
	if err != nil {
		t.Fatalf("parse structured filters: %v", err)
	}

	expected := []intmodel.EventFilter{{
		EventTypes: []string{"com.clario360.cyber.alert.created"},
		Severities: []string{"critical"},
	}}
	if !reflect.DeepEqual(filters, expected) {
		t.Fatalf("unexpected filters: got %#v want %#v", filters, expected)
	}
}

func TestParseEventFiltersLegacyStringArray(t *testing.T) {
	filters, err := parseEventFilters([]byte(`["com.clario360.cyber.alert.created","com.clario360.cyber.cti.alert.critical-threat"]`))
	if err != nil {
		t.Fatalf("parse legacy string array: %v", err)
	}

	expected := []intmodel.EventFilter{{
		EventTypes: []string{
			"com.clario360.cyber.alert.created",
			"com.clario360.cyber.cti.alert.critical-threat",
		},
	}}
	if !reflect.DeepEqual(filters, expected) {
		t.Fatalf("unexpected filters: got %#v want %#v", filters, expected)
	}
}

func TestParseEventFiltersLegacySingleString(t *testing.T) {
	filters, err := parseEventFilters([]byte(`"com.clario360.cyber.alert.created"`))
	if err != nil {
		t.Fatalf("parse legacy single string: %v", err)
	}

	expected := []intmodel.EventFilter{{
		EventTypes: []string{"com.clario360.cyber.alert.created"},
	}}
	if !reflect.DeepEqual(filters, expected) {
		t.Fatalf("unexpected filters: got %#v want %#v", filters, expected)
	}
}
