package service

import (
	"testing"

	"github.com/clario360/platform/internal/acta/model"
)

func TestDetermineVoteResultUnanimous(t *testing.T) {
	got, err := determineVoteResult(model.VoteTypeUnanimous, 8, 0, 0)
	if err != nil {
		t.Fatalf("determineVoteResult returned error: %v", err)
	}
	if got != model.VoteResultApproved {
		t.Fatalf("determineVoteResult unanimous = %s, want %s", got, model.VoteResultApproved)
	}
}

func TestDetermineVoteResultMajority(t *testing.T) {
	got, err := determineVoteResult(model.VoteTypeMajority, 6, 4, 1)
	if err != nil {
		t.Fatalf("determineVoteResult returned error: %v", err)
	}
	if got != model.VoteResultApproved {
		t.Fatalf("determineVoteResult majority = %s, want %s", got, model.VoteResultApproved)
	}
}

func TestDetermineVoteResultTwoThirds(t *testing.T) {
	got, err := determineVoteResult(model.VoteTypeTwoThirds, 7, 3, 0)
	if err != nil {
		t.Fatalf("determineVoteResult returned error: %v", err)
	}
	if got != model.VoteResultApproved {
		t.Fatalf("determineVoteResult two thirds = %s, want %s", got, model.VoteResultApproved)
	}
}

func TestDetermineVoteResultTwoThirdsFailed(t *testing.T) {
	got, err := determineVoteResult(model.VoteTypeTwoThirds, 6, 4, 0)
	if err != nil {
		t.Fatalf("determineVoteResult returned error: %v", err)
	}
	if got != model.VoteResultRejected {
		t.Fatalf("determineVoteResult two thirds failed = %s, want %s", got, model.VoteResultRejected)
	}
}

func TestDetermineVoteResultTied(t *testing.T) {
	got, err := determineVoteResult(model.VoteTypeMajority, 5, 5, 0)
	if err != nil {
		t.Fatalf("determineVoteResult returned error: %v", err)
	}
	if got != model.VoteResultTied {
		t.Fatalf("determineVoteResult tied = %s, want %s", got, model.VoteResultTied)
	}
}
