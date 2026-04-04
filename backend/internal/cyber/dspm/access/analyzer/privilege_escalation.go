package analyzer

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// PrivEscAnalyzer finds paths where an identity could escalate to higher privileges.
// It walks the permission graph to detect chains such as:
//   - admin on an application that manages roles → can create new roles
//   - write on a database + create on schema → can create new tables
//   - role membership in a group that has full_control → inherits everything
//
// MITRE: T1068 (Exploitation for Privilege Escalation)
type PrivEscAnalyzer struct {
	repo   MappingProvider
	logger zerolog.Logger
}

// NewPrivEscAnalyzer creates a new privilege escalation path finder.
func NewPrivEscAnalyzer(repo MappingProvider, logger zerolog.Logger) *PrivEscAnalyzer {
	return &PrivEscAnalyzer{
		repo:   repo,
		logger: logger.With().Str("analyzer", "privilege_escalation").Logger(),
	}
}

// FindPaths identifies privilege escalation paths for a specific identity.
func (a *PrivEscAnalyzer) FindPaths(ctx context.Context, tenantID uuid.UUID, identityID string) ([]model.EscalationPath, error) {
	mappings, err := a.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Filter to target identity.
	var identityMappings []*model.AccessMapping
	for _, m := range mappings {
		if m.IdentityID == identityID {
			identityMappings = append(identityMappings, m)
		}
	}

	// Build permission index for this identity.
	permIndex := make(map[string]map[string]bool) // assetID → permTypes
	for _, m := range identityMappings {
		assetKey := m.DataAssetID.String()
		if permIndex[assetKey] == nil {
			permIndex[assetKey] = make(map[string]bool)
		}
		permIndex[assetKey][m.PermissionType] = true
	}

	var paths []model.EscalationPath

	for _, m := range identityMappings {
		assetKey := m.DataAssetID.String()
		assetPerms := permIndex[assetKey]

		// Pattern 1: admin on application → can create roles → escalate to full_control
		if m.PermissionType == "admin" {
			paths = append(paths, model.EscalationPath{
				SourcePermission: "admin on " + m.DataAssetName,
				IntermediateStep: "Can manage roles/permissions within " + m.DataAssetName,
				TargetEscalated:  "full_control via role creation",
				RiskLevel:        "high",
			})
		}

		// Pattern 2: write + create on same asset → can create new tables/schemas
		if assetPerms["write"] && assetPerms["create"] {
			paths = append(paths, model.EscalationPath{
				SourcePermission: "write + create on " + m.DataAssetName,
				IntermediateStep: "Can create new objects and populate with data",
				TargetEscalated:  "effective admin via object creation",
				RiskLevel:        "medium",
			})
		}

		// Pattern 3: role membership via wildcard → inherits all permissions
		if m.IsWildcard && m.PermissionSource == "role_inherited" {
			paths = append(paths, model.EscalationPath{
				SourcePermission: "wildcard grant via " + m.PermissionSource,
				IntermediateStep: "Wildcard role membership inherits all current and future permissions",
				TargetEscalated:  "full_control on " + m.DataAssetName + " and future assets",
				RiskLevel:        "critical",
			})
		}

		// Pattern 4: alter + execute → can modify stored procedures
		if assetPerms["alter"] && assetPerms["execute"] {
			paths = append(paths, model.EscalationPath{
				SourcePermission: "alter + execute on " + m.DataAssetName,
				IntermediateStep: "Can modify and execute stored procedures/functions",
				TargetEscalated:  "code execution within database context",
				RiskLevel:        "high",
			})
		}

		// Pattern 5: full_control inherited via group → any group member can escalate
		if m.PermissionType == "full_control" && m.PermissionSource == "group_inherited" {
			paths = append(paths, model.EscalationPath{
				SourcePermission: "full_control via group " + pathString(m.PermissionPath),
				IntermediateStep: "Group membership provides inherited full_control",
				TargetEscalated:  "full_control on " + m.DataAssetName,
				RiskLevel:        "critical",
			})
		}
	}

	// Deduplicate paths.
	return deduplicatePaths(paths), nil
}

// FindAllPaths finds escalation paths across all identities in a tenant.
func (a *PrivEscAnalyzer) FindAllPaths(ctx context.Context, tenantID uuid.UUID) ([]model.EscalationPath, error) {
	mappings, err := a.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Collect unique identity IDs.
	identities := make(map[string]bool)
	for _, m := range mappings {
		identities[m.IdentityID] = true
	}

	var allPaths []model.EscalationPath
	for id := range identities {
		paths, err := a.FindPaths(ctx, tenantID, id)
		if err != nil {
			a.logger.Warn().Err(err).Str("identity", id).Msg("failed to find escalation paths")
			continue
		}
		allPaths = append(allPaths, paths...)
	}

	return deduplicatePaths(allPaths), nil
}

func pathString(path []string) string {
	if len(path) == 0 {
		return "direct"
	}
	result := path[0]
	for _, p := range path[1:] {
		result += " → " + p
	}
	return result
}

func deduplicatePaths(paths []model.EscalationPath) []model.EscalationPath {
	seen := make(map[string]bool)
	var result []model.EscalationPath
	for _, p := range paths {
		key := p.SourcePermission + "|" + p.TargetEscalated
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, p)
	}
	return result
}
