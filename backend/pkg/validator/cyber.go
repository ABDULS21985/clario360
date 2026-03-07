package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/clario360/platform/internal/cyber/model"
)

// alphanumDashRe matches strings containing only alphanumerics, hyphens, and underscores.
// Used for asset tag validation to prevent injection via tag values.
var alphanumDashRe = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// RegisterCyberValidators registers all cyber-service-specific struct tag validators
// with the provided validator instance. Must be called once at service startup before
// any DTO validation occurs.
func RegisterCyberValidators(v *validator.Validate) error {
	registrations := []struct {
		tag string
		fn  validator.Func
	}{
		{"asset_type", validateAssetType},
		{"criticality", validateCriticality},
		{"asset_status", validateAssetStatus},
		{"relationship_type", validateRelationshipType},
		{"alphanumdash", validateAlphanumDash},
	}
	for _, r := range registrations {
		if err := v.RegisterValidation(r.tag, r.fn); err != nil {
			return err
		}
	}
	return nil
}

func validateAssetType(fl validator.FieldLevel) bool {
	return model.AssetType(fl.Field().String()).IsValid()
}

func validateCriticality(fl validator.FieldLevel) bool {
	return model.Criticality(fl.Field().String()).IsValid()
}

func validateAssetStatus(fl validator.FieldLevel) bool {
	return model.AssetStatus(fl.Field().String()).IsValid()
}

func validateRelationshipType(fl validator.FieldLevel) bool {
	return model.RelationshipType(fl.Field().String()).IsValid()
}

// validateAlphanumDash accepts alphanumeric characters, hyphens, and underscores only.
func validateAlphanumDash(fl validator.FieldLevel) bool {
	return alphanumDashRe.MatchString(fl.Field().String())
}
