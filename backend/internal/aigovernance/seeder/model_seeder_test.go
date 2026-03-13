package seeder

import (
	"testing"

	"github.com/google/uuid"
)

func TestOptionalOwnerUserID(t *testing.T) {
	if got := optionalOwnerUserID(uuid.Nil); got != nil {
		t.Fatalf("optionalOwnerUserID(uuid.Nil) = %v, want nil", *got)
	}

	userID := uuid.New()
	got := optionalOwnerUserID(userID)
	if got == nil {
		t.Fatal("optionalOwnerUserID(userID) = nil, want pointer")
	}
	if *got != userID {
		t.Fatalf("optionalOwnerUserID(userID) = %s, want %s", *got, userID)
	}
}
