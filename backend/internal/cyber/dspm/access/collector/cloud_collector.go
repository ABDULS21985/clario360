package collector

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// CloudCollector extracts permission mappings from cloud IAM policies stored
// in each asset's metadata.cloud_iam_policies JSONB field.
type CloudCollector struct {
	logger zerolog.Logger
}

// NewCloudCollector creates a cloud IAM permission collector.
func NewCloudCollector(logger zerolog.Logger) *CloudCollector {
	return &CloudCollector{
		logger: logger.With().Str("collector", "cloud").Logger(),
	}
}

func (c *CloudCollector) Name() string { return "cloud_iam" }

// CollectPermissions iterates over assets that have cloud_iam_policies metadata
// and extracts identity → asset permission mappings. It handles wildcard policies
// (resource: "*") and condition-based policies.
func (c *CloudCollector) CollectPermissions(ctx context.Context, tenantID uuid.UUID, assets []*cybermodel.DSPMDataAsset) ([]RawPermission, error) {
	var results []RawPermission

	for _, asset := range assets {
		if asset.Metadata == nil {
			continue
		}
		policiesRaw, ok := asset.Metadata["cloud_iam_policies"]
		if !ok {
			continue
		}

		policies, ok := policiesRaw.([]interface{})
		if !ok {
			continue
		}

		for _, policyRaw := range policies {
			policy, ok := policyRaw.(map[string]interface{})
			if !ok {
				continue
			}

			principal := stringFromMap(policy, "principal")
			if principal == "" {
				principal = stringFromMap(policy, "identity")
			}
			if principal == "" {
				continue
			}

			actions := extractStringList(policy, "actions")
			if len(actions) == 0 {
				action := stringFromMap(policy, "action")
				if action != "" {
					actions = []string{action}
				}
			}

			resource := stringFromMap(policy, "resource")
			isWildcard := resource == "*" || resource == ""
			effect := stringFromMap(policy, "effect")
			if effect != "" && strings.ToLower(effect) == "deny" {
				continue
			}

			identityType := inferCloudIdentityType(principal)

			for _, action := range actions {
				permType := mapCloudAction(action)
				results = append(results, RawPermission{
					IdentityType:       identityType,
					IdentityID:         principal,
					IdentityName:       principal,
					IdentitySource:     "cloud_iam",
					DataAssetID:        asset.ID,
					DataAssetName:      asset.AssetName,
					DataClassification: asset.DataClassification,
					PermissionType:     permType,
					PermissionSource:   "policy_inherited",
					PermissionPath:     []string{stringFromMap(policy, "policy_name")},
					IsWildcard:         isWildcard || action == "*",
				})
			}
		}
	}

	return results, nil
}

// inferCloudIdentityType determines the identity type from a cloud principal string.
func inferCloudIdentityType(principal string) string {
	lower := strings.ToLower(principal)
	switch {
	case strings.Contains(lower, "serviceaccount") || strings.Contains(lower, "service-account"):
		return "service_account"
	case strings.Contains(lower, "group"):
		return "group"
	case strings.Contains(lower, "role"):
		return "role"
	default:
		return "user"
	}
}

// mapCloudAction converts a cloud IAM action to our normalized permission type.
func mapCloudAction(action string) string {
	lower := strings.ToLower(action)
	switch {
	case lower == "*" || strings.Contains(lower, "fullcontrol") || strings.Contains(lower, "full_control"):
		return "full_control"
	case strings.Contains(lower, "admin"):
		return "admin"
	case strings.Contains(lower, "delete") || strings.Contains(lower, "remove"):
		return "delete"
	case strings.Contains(lower, "write") || strings.Contains(lower, "put") || strings.Contains(lower, "update") || strings.Contains(lower, "create"):
		return "write"
	case strings.Contains(lower, "execute") || strings.Contains(lower, "invoke"):
		return "execute"
	case strings.Contains(lower, "alter") || strings.Contains(lower, "modify"):
		return "alter"
	default:
		return "read"
	}
}

func stringFromMap(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func extractStringList(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
