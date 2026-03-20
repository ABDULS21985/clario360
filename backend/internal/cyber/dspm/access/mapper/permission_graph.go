package mapper

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// MappingLister loads active access mappings for graph construction.
type MappingLister interface {
	ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.AccessMapping, error)
}

// PermissionGraph builds a directed acyclic graph of permission inheritance.
// Root nodes: identities (users, service accounts). Intermediate: roles, groups.
// Leaf nodes: data assets. Edges: permission grants with type labels.
type PermissionGraph struct {
	repo   MappingLister
	logger zerolog.Logger
}

// NewPermissionGraph creates a new graph builder.
func NewPermissionGraph(repo MappingLister, logger zerolog.Logger) *PermissionGraph {
	return &PermissionGraph{
		repo:   repo,
		logger: logger.With().Str("component", "permission_graph").Logger(),
	}
}

// BuildGraph constructs the full permission inheritance graph for a tenant.
// Returns a forest of root nodes (one per identity).
func (g *PermissionGraph) BuildGraph(ctx context.Context, tenantID uuid.UUID) ([]*model.PermissionNode, error) {
	mappings, err := g.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Group mappings by identity.
	byIdentity := make(map[string][]*model.AccessMapping)
	for _, m := range mappings {
		key := m.IdentityType + "|" + m.IdentityID
		byIdentity[key] = append(byIdentity[key], m)
	}

	var roots []*model.PermissionNode
	for key, identityMappings := range byIdentity {
		if len(identityMappings) == 0 {
			continue
		}
		first := identityMappings[0]
		identityNode := &model.PermissionNode{
			Type: first.IdentityType,
			ID:   first.IdentityID,
			Name: first.IdentityName,
		}

		// Group by permission path to build intermediate nodes.
		pathGroups := make(map[string]*model.PermissionNode)
		for _, m := range identityMappings {
			// Build intermediate nodes from permission path.
			parent := identityNode
			for _, pathElement := range m.PermissionPath {
				if pathElement == "" {
					continue
				}
				pathKey := key + "|" + pathElement
				if existing, ok := pathGroups[pathKey]; ok {
					parent = existing
				} else {
					intermediate := &model.PermissionNode{
						Type: "role",
						ID:   pathElement,
						Name: pathElement,
					}
					parent.Children = append(parent.Children, intermediate)
					pathGroups[pathKey] = intermediate
					parent = intermediate
				}
			}

			// Leaf node: the data asset with permission type.
			assetNode := &model.PermissionNode{
				Type: "asset",
				ID:   m.DataAssetID.String(),
				Name: m.DataAssetName + " [" + m.PermissionType + "]",
			}
			parent.Children = append(parent.Children, assetNode)
		}

		roots = append(roots, identityNode)
	}

	return roots, nil
}

// BuildGraphForIdentity constructs the permission graph for a single identity.
func (g *PermissionGraph) BuildGraphForIdentity(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.PermissionNode, error) {
	mappings, err := g.repo.ListActiveByTenant(ctx, tenantID)
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

	if len(identityMappings) == 0 {
		return nil, nil
	}

	first := identityMappings[0]
	root := &model.PermissionNode{
		Type: first.IdentityType,
		ID:   first.IdentityID,
		Name: first.IdentityName,
	}

	pathGroups := make(map[string]*model.PermissionNode)
	for _, m := range identityMappings {
		parent := root
		for _, pathElement := range m.PermissionPath {
			if pathElement == "" {
				continue
			}
			pathKey := pathElement
			if existing, ok := pathGroups[pathKey]; ok {
				parent = existing
			} else {
				intermediate := &model.PermissionNode{
					Type: "role",
					ID:   pathElement,
					Name: pathElement,
				}
				parent.Children = append(parent.Children, intermediate)
				pathGroups[pathKey] = intermediate
				parent = intermediate
			}
		}

		assetNode := &model.PermissionNode{
			Type: "asset",
			ID:   m.DataAssetID.String(),
			Name: m.DataAssetName + " [" + m.PermissionType + "]",
		}
		parent.Children = append(parent.Children, assetNode)
	}

	return root, nil
}
