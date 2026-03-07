package analyzer

import (
	"testing"
	"time"
)

func TestExtract_PartyNames(t *testing.T) {
	extractor := NewEntityExtractor()
	parties, _, _ := extractor.Extract("This agreement is entered into between Company A and Company B.\nEffective as of January 1, 2026.")
	if len(parties) != 2 {
		t.Fatalf("len(parties) = %d, want 2", len(parties))
	}
	if parties[0].Name != "Company A" || parties[1].Name != "Company B" {
		t.Fatalf("unexpected parties: %+v", parties)
	}
}

func TestExtract_Dates(t *testing.T) {
	extractor := NewEntityExtractor()
	_, dates, _ := extractor.Extract("This agreement is effective as of January 1, 2026. Expiry date is 2026-12-31.")
	if len(dates) == 0 || dates[0].Value == nil {
		t.Fatalf("expected at least one parsed date, got %+v", dates)
	}
	want := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !dates[0].Value.Equal(want) {
		t.Fatalf("first date = %s, want %s", dates[0].Value, want)
	}
}

func TestExtract_Amounts(t *testing.T) {
	extractor := NewEntityExtractor()
	_, _, amounts := extractor.Extract("The total value of SAR 500,000 applies under this agreement.")
	if len(amounts) != 1 {
		t.Fatalf("len(amounts) = %d, want 1", len(amounts))
	}
	if amounts[0].Currency != "SAR" || amounts[0].Value != 500000 {
		t.Fatalf("unexpected amount extraction: %+v", amounts[0])
	}
}
