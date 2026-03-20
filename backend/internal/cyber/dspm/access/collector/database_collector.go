package collector

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// DatabaseCollector extracts permission mappings from PostgreSQL system catalogs.
// It queries pg_roles, information_schema.role_table_grants, and pg_auth_members
// to build a complete picture of which identities can access which database assets.
type DatabaseCollector struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewDatabaseCollector creates a database permission collector.
func NewDatabaseCollector(db *pgxpool.Pool, logger zerolog.Logger) *DatabaseCollector {
	return &DatabaseCollector{
		db:     db,
		logger: logger.With().Str("collector", "database").Logger(),
	}
}

func (c *DatabaseCollector) Name() string { return "database" }

// CollectPermissions queries the database system catalogs to extract role→table permission
// mappings. It handles role inheritance chains via recursive traversal of pg_auth_members.
func (c *DatabaseCollector) CollectPermissions(ctx context.Context, tenantID uuid.UUID, assets []*cybermodel.DSPMDataAsset) ([]RawPermission, error) {
	// Filter to database-type assets only.
	dbAssets := make(map[string]*cybermodel.DSPMDataAsset) // key: asset_name
	for _, a := range assets {
		if a.DatabaseType != nil && *a.DatabaseType != "" {
			dbAssets[strings.ToLower(a.AssetName)] = a
		}
	}
	if len(dbAssets) == 0 {
		return nil, nil
	}

	// Collect role-to-table grants from information_schema.
	grants, err := c.collectTableGrants(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to collect table grants")
		return nil, nil
	}

	// Collect role inheritance chains from pg_auth_members.
	inheritance, err := c.collectRoleInheritance(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to collect role inheritance")
		// Non-fatal: we can still produce direct grants.
	}

	var results []RawPermission
	for _, grant := range grants {
		// Match grant table to a tracked data asset.
		asset, found := dbAssets[strings.ToLower(grant.tableName)]
		if !found {
			continue
		}

		permType := mapPostgresPrivilege(grant.privilegeType)
		permSource := "direct_grant"
		var permPath []string
		isWildcard := grant.tableName == "*" || grant.tableSchema == "*"

		// Check if this role is inherited rather than directly assigned.
		if chain, ok := inheritance[grant.grantee]; ok && len(chain) > 0 {
			permSource = "role_inherited"
			permPath = chain
		}

		results = append(results, RawPermission{
			IdentityType:       "role",
			IdentityID:         grant.grantee,
			IdentityName:       grant.grantee,
			IdentitySource:     "database",
			DataAssetID:        asset.ID,
			DataAssetName:      asset.AssetName,
			DataClassification: asset.DataClassification,
			PermissionType:     permType,
			PermissionSource:   permSource,
			PermissionPath:     permPath,
			IsWildcard:         isWildcard,
		})
	}

	return results, nil
}

type tableGrant struct {
	grantee       string
	tableSchema   string
	tableName     string
	privilegeType string
}

func (c *DatabaseCollector) collectTableGrants(ctx context.Context) ([]tableGrant, error) {
	rows, err := c.db.Query(ctx, `
		SELECT grantee, table_schema, table_name, privilege_type
		FROM information_schema.role_table_grants
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []tableGrant
	for rows.Next() {
		var g tableGrant
		if err := rows.Scan(&g.grantee, &g.tableSchema, &g.tableName, &g.privilegeType); err != nil {
			continue
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

// collectRoleInheritance builds a map: roleName → chain of parent roles.
// Uses a recursive approach on pg_auth_members to follow the inheritance tree.
func (c *DatabaseCollector) collectRoleInheritance(ctx context.Context) (map[string][]string, error) {
	rows, err := c.db.Query(ctx, `
		WITH RECURSIVE role_tree AS (
			SELECT
				m.member::regrole::text AS member,
				m.roleid::regrole::text AS parent,
				ARRAY[m.roleid::regrole::text] AS chain
			FROM pg_auth_members m
			UNION ALL
			SELECT
				rt.member,
				m.roleid::regrole::text,
				rt.chain || m.roleid::regrole::text
			FROM role_tree rt
			JOIN pg_auth_members m ON m.member = (SELECT oid FROM pg_roles WHERE rolname = rt.parent)
			WHERE array_length(rt.chain, 1) < 10
		)
		SELECT member, chain FROM role_tree
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var member string
		var chain []string
		if err := rows.Scan(&member, &chain); err != nil {
			continue
		}
		// Keep the longest chain (most complete inheritance path).
		if existing, ok := result[member]; !ok || len(chain) > len(existing) {
			result[member] = chain
		}
	}
	return result, rows.Err()
}

// mapPostgresPrivilege converts a PostgreSQL privilege to our normalized permission type.
func mapPostgresPrivilege(privilege string) string {
	switch strings.ToUpper(privilege) {
	case "SELECT":
		return "read"
	case "INSERT":
		return "write"
	case "UPDATE":
		return "write"
	case "DELETE":
		return "delete"
	case "TRUNCATE":
		return "delete"
	case "REFERENCES":
		return "read"
	case "TRIGGER":
		return "execute"
	case "CREATE":
		return "create"
	case "ALL", "ALL PRIVILEGES":
		return "full_control"
	default:
		return "read"
	}
}
