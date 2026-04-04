package collector

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// ApplicationCollector extracts permission mappings from the platform's own IAM
// service. It queries users → roles → permissions and maps platform permissions
// (e.g. "cyber:read", "dspm:admin") to data asset access.
type ApplicationCollector struct {
	platformDB *pgxpool.Pool
	logger     zerolog.Logger
}

// NewApplicationCollector creates a collector for platform IAM role assignments.
func NewApplicationCollector(platformDB *pgxpool.Pool, logger zerolog.Logger) *ApplicationCollector {
	return &ApplicationCollector{
		platformDB: platformDB,
		logger:     logger.With().Str("collector", "application").Logger(),
	}
}

func (c *ApplicationCollector) Name() string { return "application" }

// CollectPermissions queries the platform_core database for user→role→permission
// mappings and maps them to data asset access levels.
func (c *ApplicationCollector) CollectPermissions(ctx context.Context, tenantID uuid.UUID, assets []*cybermodel.DSPMDataAsset) ([]RawPermission, error) {
	if c.platformDB == nil {
		return nil, nil
	}

	// Query users and their role-based permissions for this tenant.
	rows, err := c.platformDB.Query(ctx, `
		SELECT
			u.id,
			u.email,
			u.first_name || ' ' || u.last_name AS display_name,
			r.name AS role_name,
			p.permission
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		JOIN role_permissions rp ON rp.role_id = r.id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE u.tenant_id = $1
		  AND u.deleted_at IS NULL
	`, tenantID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to query platform user permissions")
		return nil, nil
	}
	defer rows.Close()

	type userPerm struct {
		userID      string
		email       string
		displayName string
		roleName    string
		permission  string
	}

	var perms []userPerm
	for rows.Next() {
		var p userPerm
		if err := rows.Scan(&p.userID, &p.email, &p.displayName, &p.roleName, &p.permission); err != nil {
			continue
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var results []RawPermission
	for _, perm := range perms {
		permType, appliesTo := mapPlatformPermission(perm.permission)
		if appliesTo == "" && permType != "full_control" {
			continue
		}

		// Wildcard permission ("*") grants full_control to everything.
		if perm.permission == "*" || permType == "full_control" {
			for _, asset := range assets {
				results = append(results, RawPermission{
					IdentityType:       "user",
					IdentityID:         perm.userID,
					IdentityName:       perm.displayName,
					IdentitySource:     "application",
					DataAssetID:        asset.ID,
					DataAssetName:      asset.AssetName,
					DataClassification: asset.DataClassification,
					PermissionType:     "full_control",
					PermissionSource:   "role_inherited",
					PermissionPath:     []string{perm.roleName},
					IsWildcard:         true,
				})
			}
			continue
		}

		// Map domain-specific permissions to matching assets.
		for _, asset := range assets {
			if !assetMatchesDomain(asset, appliesTo) {
				continue
			}
			results = append(results, RawPermission{
				IdentityType:       "user",
				IdentityID:         perm.userID,
				IdentityName:       perm.displayName,
				IdentitySource:     "application",
				DataAssetID:        asset.ID,
				DataAssetName:      asset.AssetName,
				DataClassification: asset.DataClassification,
				PermissionType:     permType,
				PermissionSource:   "role_inherited",
				PermissionPath:     []string{perm.roleName, perm.permission},
				IsWildcard:         false,
			})
		}
	}

	return results, nil
}

// mapPlatformPermission converts a platform permission string to (permission_type, domain).
// Example: "dspm:read" → ("read", "dspm"), "cyber:admin" → ("admin", "cyber"), "*" → ("full_control", "")
func mapPlatformPermission(permission string) (string, string) {
	if permission == "*" {
		return "full_control", ""
	}
	parts := strings.SplitN(permission, ":", 2)
	if len(parts) != 2 {
		return "read", ""
	}
	domain := parts[0]
	action := parts[1]
	switch action {
	case "admin":
		return "admin", domain
	case "write", "create", "update":
		return "write", domain
	case "delete":
		return "delete", domain
	case "read":
		return "read", domain
	default:
		return "read", domain
	}
}

// assetMatchesDomain checks if a data asset belongs to a given permission domain.
func assetMatchesDomain(asset *cybermodel.DSPMDataAsset, domain string) bool {
	switch domain {
	case "dspm", "cyber", "data":
		return true
	case "assets":
		return true
	default:
		return false
	}
}
