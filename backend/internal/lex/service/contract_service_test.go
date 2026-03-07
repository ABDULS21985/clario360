package service

import (
	"testing"

	"github.com/clario360/platform/internal/lex/model"
)

func TestStatusTransition_DraftToReview(t *testing.T) {
	if err := ValidateContractTransition(string(model.ContractStatusDraft), string(model.ContractStatusInternalReview)); err != nil {
		t.Fatalf("ValidateContractTransition() error = %v", err)
	}
}

func TestStatusTransition_ActiveToExpired(t *testing.T) {
	if err := ValidateContractTransition(string(model.ContractStatusActive), string(model.ContractStatusExpired)); err != nil {
		t.Fatalf("ValidateContractTransition() error = %v", err)
	}
}

func TestStatusTransition_DraftToActive(t *testing.T) {
	if err := ValidateContractTransition(string(model.ContractStatusDraft), string(model.ContractStatusActive)); err == nil {
		t.Fatal("ValidateContractTransition() error = nil, want invalid transition error")
	}
}

func TestStatusTransition_AllValidPaths(t *testing.T) {
	tests := []struct {
		current model.ContractStatus
		next    model.ContractStatus
	}{
		{model.ContractStatusDraft, model.ContractStatusInternalReview},
		{model.ContractStatusDraft, model.ContractStatusCancelled},
		{model.ContractStatusInternalReview, model.ContractStatusLegalReview},
		{model.ContractStatusInternalReview, model.ContractStatusDraft},
		{model.ContractStatusLegalReview, model.ContractStatusNegotiation},
		{model.ContractStatusLegalReview, model.ContractStatusInternalReview},
		{model.ContractStatusLegalReview, model.ContractStatusDraft},
		{model.ContractStatusNegotiation, model.ContractStatusPendingSignature},
		{model.ContractStatusNegotiation, model.ContractStatusCancelled},
		{model.ContractStatusNegotiation, model.ContractStatusDraft},
		{model.ContractStatusPendingSignature, model.ContractStatusActive},
		{model.ContractStatusPendingSignature, model.ContractStatusCancelled},
		{model.ContractStatusActive, model.ContractStatusSuspended},
		{model.ContractStatusActive, model.ContractStatusTerminated},
		{model.ContractStatusActive, model.ContractStatusExpired},
		{model.ContractStatusActive, model.ContractStatusRenewed},
		{model.ContractStatusSuspended, model.ContractStatusActive},
		{model.ContractStatusSuspended, model.ContractStatusTerminated},
		{model.ContractStatusExpired, model.ContractStatusRenewed},
	}
	for _, tc := range tests {
		if err := ValidateContractTransition(string(tc.current), string(tc.next)); err != nil {
			t.Fatalf("ValidateContractTransition(%s, %s) error = %v", tc.current, tc.next, err)
		}
	}
}
