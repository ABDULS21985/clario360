package hash

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/audit/model"
)

// GenesisHash is the initial previous hash for the first entry in a tenant's chain.
const GenesisHash = "GENESIS"

// ComputeEntryHash computes the SHA-256 hash for an audit entry.
//
// The hash input is a deterministic concatenation of fields delimited by '|':
//
//	id|tenant_id|user_id|service|action|resource_type|resource_id|old_value|new_value|created_at_unix_nano|previous_hash
//
// Rules:
//   - *string nil → empty string ""
//   - json.RawMessage nil → empty string ""
//   - json.RawMessage present → compact JSON (no whitespace variance)
//   - time.Time → UnixNano() as decimal string
//   - Output: lowercase hex-encoded SHA-256
//
// This function is PURE — no I/O, no randomness, deterministic for identical inputs.
func ComputeEntryHash(entry *model.AuditEntry, previousHash string) string {
	var b strings.Builder

	b.WriteString(entry.ID)
	b.WriteByte('|')
	b.WriteString(entry.TenantID)
	b.WriteByte('|')

	if entry.UserID != nil {
		b.WriteString(*entry.UserID)
	}
	b.WriteByte('|')

	b.WriteString(entry.Service)
	b.WriteByte('|')
	b.WriteString(entry.Action)
	b.WriteByte('|')
	b.WriteString(entry.ResourceType)
	b.WriteByte('|')
	b.WriteString(entry.ResourceID)
	b.WriteByte('|')

	b.WriteString(compactJSON(entry.OldValue))
	b.WriteByte('|')
	b.WriteString(compactJSON(entry.NewValue))
	b.WriteByte('|')

	b.WriteString(fmt.Sprintf("%d", entry.CreatedAt.UnixNano()))
	b.WriteByte('|')
	b.WriteString(previousHash)

	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("%x", sum[:])
}

// compactJSON returns a compact JSON string for the given raw message.
// Returns empty string for nil/empty input.
func compactJSON(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		// If compaction fails, use the raw bytes.
		return string(data)
	}
	return buf.String()
}
