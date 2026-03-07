package analytics

import (
	"fmt"
	"slices"
	"strings"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/data/model"
)

type ValidationContext struct {
	ColumnsAccessed      []string
	PIIColumnsAccessed   []string
	UserHasPIIPermission bool
	Limit                int
}

func ValidateQuery(query *model.AnalyticsQuery, dataModel *model.DataModel, userPermissions []string) error {
	_, err := AnalyzeQuery(query, dataModel, userPermissions, false)
	return err
}

func AnalyzeQuery(query *model.AnalyticsQuery, dataModel *model.DataModel, userPermissions []string, explore bool) (*ValidationContext, error) {
	if dataModel == nil {
		return nil, fmt.Errorf("data model is required")
	}
	if dataModel.Status != model.DataModelStatusActive {
		return nil, fmt.Errorf("model %q is not active", dataModel.Name)
	}
	if query == nil {
		query = &model.AnalyticsQuery{}
	}

	fieldMap := make(map[string]model.ModelField, len(dataModel.SchemaDefinition))
	allColumns := make([]string, 0, len(dataModel.SchemaDefinition))
	piiColumns := make([]string, 0)
	for _, field := range dataModel.SchemaDefinition {
		key := strings.ToLower(field.Name)
		fieldMap[key] = field
		allColumns = append(allColumns, field.Name)
		if field.PIIType != "" {
			piiColumns = append(piiColumns, field.Name)
		}
	}

	if err := validateClassificationAccess(dataModel.DataClassification, userPermissions); err != nil {
		return nil, err
	}

	columnsAccessed := make([]string, 0)
	if len(query.Aggregations) == 0 {
		if len(query.Columns) == 0 {
			query.Columns = append([]string(nil), allColumns...)
		}
		for _, column := range query.Columns {
			if _, ok := fieldMap[strings.ToLower(column)]; !ok {
				return nil, fmt.Errorf("column %q does not exist in model %q", column, dataModel.Name)
			}
			columnsAccessed = append(columnsAccessed, column)
		}
	}

	for _, filter := range query.Filters {
		if _, ok := fieldMap[strings.ToLower(filter.Column)]; !ok {
			return nil, fmt.Errorf("column %q does not exist in model %q", filter.Column, dataModel.Name)
		}
		if err := validateFilter(filter); err != nil {
			return nil, err
		}
		columnsAccessed = append(columnsAccessed, filter.Column)
	}
	for _, column := range query.GroupBy {
		if _, ok := fieldMap[strings.ToLower(column)]; !ok {
			return nil, fmt.Errorf("column %q does not exist in model %q", column, dataModel.Name)
		}
		columnsAccessed = append(columnsAccessed, column)
	}
	for _, agg := range query.Aggregations {
		if agg.Column != "" {
			if _, ok := fieldMap[strings.ToLower(agg.Column)]; !ok {
				return nil, fmt.Errorf("column %q does not exist in model %q", agg.Column, dataModel.Name)
			}
			columnsAccessed = append(columnsAccessed, agg.Column)
		}
		switch strings.ToLower(strings.TrimSpace(agg.Function)) {
		case "count", "sum", "avg", "min", "max":
		default:
			return nil, fmt.Errorf("unsupported aggregation %q", agg.Function)
		}
	}
	for _, order := range query.OrderBy {
		if _, ok := fieldMap[strings.ToLower(order.Column)]; !ok && order.Column != "" {
			if !matchesAggregationAlias(order.Column, query.Aggregations) {
				return nil, fmt.Errorf("column %q does not exist in model %q", order.Column, dataModel.Name)
			}
		}
		if dir := strings.ToLower(strings.TrimSpace(order.Direction)); dir != "" && dir != "asc" && dir != "desc" {
			return nil, fmt.Errorf("invalid order direction %q", order.Direction)
		}
		columnsAccessed = append(columnsAccessed, order.Column)
	}

	if query.Limit <= 0 {
		if explore {
			query.Limit = 100
		} else {
			query.Limit = 1000
		}
	}
	if explore && query.Limit > 100 {
		query.Limit = 100
	}
	if query.Limit > 10000 {
		return nil, fmt.Errorf("query limit must not exceed 10000")
	}
	if query.Offset < 0 {
		return nil, fmt.Errorf("query offset must not be negative")
	}

	columnsAccessed = uniqueStrings(columnsAccessed)
	piiAccessed := make([]string, 0)
	for _, column := range columnsAccessed {
		field, ok := fieldMap[strings.ToLower(column)]
		if ok && field.PIIType != "" {
			piiAccessed = append(piiAccessed, field.Name)
		}
	}

	return &ValidationContext{
		ColumnsAccessed:      columnsAccessed,
		PIIColumnsAccessed:   uniqueStrings(piiAccessed),
		UserHasPIIPermission: hasPermission(userPermissions, auth.PermDataPII),
		Limit:                query.Limit,
	}, nil
}

func validateClassificationAccess(classification model.DataClassification, permissions []string) error {
	switch classification {
	case model.DataClassificationRestricted:
		if !hasPermission(permissions, auth.PermDataRestricted) {
			return fmt.Errorf("you do not have permission to query restricted data")
		}
	case model.DataClassificationConfidential:
		if !hasAnyPermission(permissions, auth.PermDataConfidential, auth.PermDataRestricted) {
			return fmt.Errorf("you do not have permission to query confidential data")
		}
	}
	return nil
}

func validateFilter(filter model.AnalyticsFilter) error {
	operator := strings.ToLower(strings.TrimSpace(filter.Operator))
	switch operator {
	case "eq", "neq", "gt", "gte", "lt", "lte", "like", "ilike":
		if filter.Value == nil {
			return fmt.Errorf("operator %q requires a value", operator)
		}
	case "between":
		switch value := filter.Value.(type) {
		case []any:
			if len(value) != 2 {
				return fmt.Errorf("between requires a two-value array")
			}
		default:
			return fmt.Errorf("between requires a two-value array")
		}
	case "in", "not_in":
		switch value := filter.Value.(type) {
		case []any:
			if len(value) == 0 {
				return fmt.Errorf("%s requires a non-empty array", operator)
			}
		case []string:
			if len(value) == 0 {
				return fmt.Errorf("%s requires a non-empty array", operator)
			}
		default:
			return fmt.Errorf("%s requires an array value", operator)
		}
	case "is_null", "is_not_null":
	default:
		return fmt.Errorf("invalid operator %q", filter.Operator)
	}
	return nil
}

func matchesAggregationAlias(value string, aggregations []model.AnalyticsAggregation) bool {
	for _, agg := range aggregations {
		if strings.EqualFold(strings.TrimSpace(agg.Alias), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, value)
	}
	slices.Sort(unique)
	return unique
}

func hasPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == auth.PermAdminAll || permission == required {
			return true
		}
		if strings.HasSuffix(permission, ":*") && strings.HasPrefix(required, strings.TrimSuffix(permission, "*")) {
			return true
		}
	}
	return false
}

func hasAnyPermission(permissions []string, required ...string) bool {
	for _, permission := range required {
		if hasPermission(permissions, permission) {
			return true
		}
	}
	return false
}
