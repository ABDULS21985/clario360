package service

import (
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
)

func TestWidgetPosition_Valid(t *testing.T) {
	err := ValidateWidgetLayout([]model.Widget{
		{ID: uuid.New(), Position: model.WidgetPosition{X: 0, Y: 0, W: 6, H: 2}},
		{ID: uuid.New(), Position: model.WidgetPosition{X: 6, Y: 0, W: 6, H: 2}},
	})
	if err != nil {
		t.Fatalf("expected valid layout, got %v", err)
	}
}

func TestWidgetPosition_Overlap(t *testing.T) {
	err := ValidateWidgetLayout([]model.Widget{
		{ID: uuid.New(), Position: model.WidgetPosition{X: 0, Y: 0, W: 6, H: 2}},
		{ID: uuid.New(), Position: model.WidgetPosition{X: 0, Y: 0, W: 6, H: 2}},
	})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestWidgetPosition_ExceedsGrid(t *testing.T) {
	err := ValidateWidgetLayout([]model.Widget{
		{ID: uuid.New(), Position: model.WidgetPosition{X: 10, Y: 0, W: 4, H: 2}},
	})
	if err == nil {
		t.Fatal("expected grid boundary error")
	}
}

func TestWidgetPosition_ValidEdge(t *testing.T) {
	err := ValidateWidgetLayout([]model.Widget{
		{ID: uuid.New(), Position: model.WidgetPosition{X: 9, Y: 0, W: 3, H: 2}},
	})
	if err != nil {
		t.Fatalf("expected valid edge-aligned widget, got %v", err)
	}
}
