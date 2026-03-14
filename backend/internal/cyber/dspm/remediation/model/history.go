package model

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HistoryAction enumerates actions recorded in remediation history.
type HistoryAction string

const (
	HistoryActionStepStarted      HistoryAction = "step_started"
	HistoryActionStepCompleted    HistoryAction = "step_completed"
	HistoryActionStepFailed       HistoryAction = "step_failed"
	HistoryActionAssigned         HistoryAction = "assigned"
	HistoryActionStatusChanged    HistoryAction = "status_changed"
	HistoryActionRolledBack       HistoryAction = "rolled_back"
	HistoryActionSLABreached      HistoryAction = "sla_breached"
	HistoryActionExceptionGranted HistoryAction = "exception_granted"
	HistoryActionNoteAdded        HistoryAction = "note_added"
	HistoryActionCreated          HistoryAction = "created"
)

// ActorType enumerates who performed a history action.
type ActorType string

const (
	ActorTypeUser         ActorType = "user"
	ActorTypeSystem       ActorType = "system"
	ActorTypePolicyEngine ActorType = "policy_engine"
	ActorTypeScheduler    ActorType = "scheduler"
)

// RemediationHistory is an entry in the tamper-evident audit trail.
type RemediationHistory struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TenantID       uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	RemediationID  uuid.UUID       `json:"remediation_id" db:"remediation_id"`
	Action         HistoryAction   `json:"action" db:"action"`
	ActorID        *uuid.UUID      `json:"actor_id,omitempty" db:"actor_id"`
	ActorType      ActorType       `json:"actor_type" db:"actor_type"`
	Details        json.RawMessage `json:"details" db:"details"`
	EntryHash      string          `json:"entry_hash" db:"entry_hash"`
	PrevHash       string          `json:"prev_hash,omitempty" db:"prev_hash"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// ComputeEntryHash computes a SHA-256 hash of the history entry for tamper detection.
// Formula: SHA-256(prev_hash + action + details + timestamp)
func ComputeEntryHash(prevHash string, action HistoryAction, details json.RawMessage, timestamp time.Time) string {
	data := fmt.Sprintf("%s|%s|%s|%s", prevHash, action, string(details), timestamp.UTC().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// VerifyChain checks that a sequence of history entries forms a valid hash chain.
func VerifyChain(entries []RemediationHistory) (bool, int) {
	for i, entry := range entries {
		expectedHash := ComputeEntryHash(entry.PrevHash, entry.Action, entry.Details, entry.CreatedAt)
		if entry.EntryHash != expectedHash {
			return false, i
		}
		if i > 0 && entry.PrevHash != entries[i-1].EntryHash {
			return false, i
		}
	}
	return true, -1
}
