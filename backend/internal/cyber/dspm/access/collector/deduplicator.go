package collector

import "fmt"

// Deduplicator removes duplicate permission records. When the same effective
// access (identity_type + identity_id + data_asset_id + permission_type) comes
// from multiple paths, it keeps the most direct (shortest permission_path).
// If paths are equal length, it keeps the most recently granted.
type Deduplicator struct{}

// NewDeduplicator creates a new permission deduplicator.
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{}
}

// Deduplicate takes a raw permission slice and returns deduplicated results.
// Key: (identity_type, identity_id, data_asset_id, permission_type).
// Tie-break: shortest permission_path wins; equal length → most recent GrantedAt wins.
func (d *Deduplicator) Deduplicate(perms []RawPermission) []RawPermission {
	seen := make(map[string]int) // key → index in result
	result := make([]RawPermission, 0, len(perms))

	for _, p := range perms {
		key := dedupKey(p)
		if idx, ok := seen[key]; ok {
			existing := result[idx]
			if shouldReplace(existing, p) {
				result[idx] = p
			}
		} else {
			seen[key] = len(result)
			result = append(result, p)
		}
	}
	return result
}

func dedupKey(p RawPermission) string {
	return fmt.Sprintf("%s|%s|%s|%s", p.IdentityType, p.IdentityID, p.DataAssetID, p.PermissionType)
}

// shouldReplace returns true if candidate should replace existing.
func shouldReplace(existing, candidate RawPermission) bool {
	existingLen := len(existing.PermissionPath)
	candidateLen := len(candidate.PermissionPath)

	// Shorter path = more direct access → keep it.
	if candidateLen < existingLen {
		return true
	}
	if candidateLen > existingLen {
		return false
	}

	// Equal path length: prefer the most recently granted.
	if candidate.GrantedAt != nil && existing.GrantedAt != nil {
		return candidate.GrantedAt.After(*existing.GrantedAt)
	}
	// If candidate has a grant time and existing doesn't, prefer candidate.
	if candidate.GrantedAt != nil {
		return true
	}
	return false
}
