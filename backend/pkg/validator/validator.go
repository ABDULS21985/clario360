package validator

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/clario360/platform/internal/cyber/model"
)

// V is the package-level validator instance.
var V *validator.Validate

// alphanumDashRe matches strings containing only letters, digits, hyphens, and underscores.
var alphanumDashRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func init() {
	V = validator.New()

	// Use JSON tag names for error messages
	V.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	_ = RegisterCustomValidators(V)
}

// RegisterCustomValidators registers all Clario-specific validators on the provided validator instance.
func RegisterCustomValidators(v *validator.Validate) error {
	if err := v.RegisterValidation("asset_type", func(fl validator.FieldLevel) bool {
		return model.AssetType(fl.Field().String()).IsValid()
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("criticality", func(fl validator.FieldLevel) bool {
		return model.Criticality(fl.Field().String()).IsValid()
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("asset_status", func(fl validator.FieldLevel) bool {
		return model.AssetStatus(fl.Field().String()).IsValid()
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("relationship_type", func(fl validator.FieldLevel) bool {
		return model.RelationshipType(fl.Field().String()).IsValid()
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("alphanumdash", func(fl validator.FieldLevel) bool {
		return alphanumDashRe.MatchString(fl.Field().String())
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("mac", func(fl validator.FieldLevel) bool {
		_, err := net.ParseMAC(fl.Field().String())
		return err == nil
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("datestr", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if _, err := time.Parse("2006-01-02", s); err == nil {
			return true
		}
		if _, err := time.Parse(time.RFC3339, s); err == nil {
			return true
		}
		return false
	}); err != nil {
		return err
	}
	return nil
}

// Validate validates a struct and returns a map of field-level errors.
func Validate(s any) map[string]string {
	err := V.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_error": err.Error()}
	}

	fields := make(map[string]string, len(validationErrors))
	for _, fe := range validationErrors {
		fields[fe.Field()] = formatError(fe)
	}
	return fields
}

// ValidateVar validates a single variable against a tag.
func ValidateVar(field any, tag string) error {
	return V.Var(field, tag)
}

func formatError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "url":
		return "must be a valid URL"
	case "asset_type":
		return "must be a valid asset type (server, endpoint, network_device, cloud_resource, iot_device, application, database, container)"
	case "criticality":
		return "must be a valid criticality level (critical, high, medium, low)"
	case "asset_status":
		return "must be a valid asset status (active, inactive, decommissioned, unknown)"
	case "relationship_type":
		return "must be a valid relationship type (hosts, runs_on, connects_to, depends_on, managed_by, backs_up, load_balances)"
	case "alphanumdash":
		return "must contain only letters, digits, hyphens, and underscores"
	case "datestr":
		return "must be a valid date (YYYY-MM-DD or RFC3339)"
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}
