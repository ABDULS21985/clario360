package discovery

import "strings"

type TypeMapping struct {
	Type    string
	Subtype string
}

func MapNativeType(nativeType string) TypeMapping {
	value := strings.ToLower(strings.TrimSpace(nativeType))

	switch {
	case value == "", value == "unknown":
		return TypeMapping{Type: "string"}
	case strings.HasPrefix(value, "_"):
		return TypeMapping{Type: "array"}
	case strings.Contains(value, "char"), strings.Contains(value, "text"), strings.Contains(value, "enum"), strings.Contains(value, "set"):
		return TypeMapping{Type: "string"}
	case strings.Contains(value, "int"), strings.Contains(value, "serial"), strings.Contains(value, "bigserial"), strings.Contains(value, "smallserial"):
		if value == "tinyint(1)" {
			return TypeMapping{Type: "boolean"}
		}
		return TypeMapping{Type: "integer"}
	case strings.Contains(value, "numeric"), strings.Contains(value, "decimal"), strings.Contains(value, "float"), strings.Contains(value, "double"), strings.Contains(value, "real"):
		return TypeMapping{Type: "float"}
	case strings.Contains(value, "bool"):
		return TypeMapping{Type: "boolean"}
	case strings.Contains(value, "timestamp"), value == "date", value == "time", value == "datetime", value == "timestamptz":
		return TypeMapping{Type: "datetime"}
	case strings.Contains(value, "json"):
		return TypeMapping{Type: "json"}
	case strings.Contains(value, "bytea"), strings.Contains(value, "blob"), strings.Contains(value, "binary"), strings.Contains(value, "varbinary"):
		return TypeMapping{Type: "binary"}
	case value == "uuid":
		return TypeMapping{Type: "string", Subtype: "uuid"}
	case value == "inet" || value == "cidr":
		return TypeMapping{Type: "string", Subtype: "ip"}
	case strings.Contains(value, "array"):
		return TypeMapping{Type: "array"}
	default:
		return TypeMapping{Type: "string"}
	}
}
