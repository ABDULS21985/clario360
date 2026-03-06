package events

import (
	"encoding/json"
	"testing"
)

func TestSerializer_Serialize_Deserialize(t *testing.T) {
	s := NewSerializer()

	original, err := NewEvent("test.event", "test-service", "tenant-1", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	data, err := s.Serialize(original)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	restored, err := s.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", restored.ID, original.ID)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: %s != %s", restored.Type, original.Type)
	}
	if restored.TenantID != original.TenantID {
		t.Errorf("TenantID mismatch: %s != %s", restored.TenantID, original.TenantID)
	}
}

func TestSerializer_Serialize_ValidationFailure(t *testing.T) {
	s := NewSerializer()
	event := &Event{} // missing required fields
	_, err := s.Serialize(event)
	if err == nil {
		t.Error("expected validation error for empty event")
	}
}

func TestSerializer_Deserialize_InvalidJSON(t *testing.T) {
	s := NewSerializer()
	_, err := s.Deserialize([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSerializer_Deserialize_MissingFields(t *testing.T) {
	s := NewSerializer()
	// Valid JSON but missing required CloudEvents fields
	_, err := s.Deserialize([]byte(`{"id":"1"}`))
	if err == nil {
		t.Error("expected validation error for incomplete event")
	}
}

func TestSerializer_SerializeData(t *testing.T) {
	s := NewSerializer()
	event, _ := NewEvent("test", "svc", "t1", nil)

	payload := map[string]int{"count": 42}
	if err := s.SerializeData(event, payload); err != nil {
		t.Fatalf("SerializeData failed: %v", err)
	}

	var result map[string]int
	if err := json.Unmarshal(event.Data, &result); err != nil {
		t.Fatalf("Unmarshal data failed: %v", err)
	}
	if result["count"] != 42 {
		t.Errorf("expected count=42, got %d", result["count"])
	}
}

func TestSerializer_DeserializeData(t *testing.T) {
	s := NewSerializer()
	event, _ := NewEvent("test", "svc", "t1", map[string]string{"name": "test"})

	var result map[string]string
	if err := s.DeserializeData(event, &result); err != nil {
		t.Fatalf("DeserializeData failed: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name=test, got %s", result["name"])
	}
}

func TestSerializer_DeserializeData_NilData(t *testing.T) {
	s := NewSerializer()
	event := &Event{Data: nil}

	var result map[string]string
	if err := s.DeserializeData(event, &result); err == nil {
		t.Error("expected error for nil data")
	}
}
