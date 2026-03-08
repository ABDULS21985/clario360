package connector

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

var (
	clickHouseNullableRe       = regexp.MustCompile(`^Nullable\((.+)\)$`)
	clickHouseLowCardinalityRe = regexp.MustCompile(`^LowCardinality\((.+)\)$`)
	clickHouseArrayRe          = regexp.MustCompile(`^Array\((.+)\)$`)
	clickHouseDecimalRe        = regexp.MustCompile(`^Decimal\(\d+\s*,\s*\d+\)$`)
	clickHouseFixedStringRe    = regexp.MustCompile(`^FixedString\(\d+\)$`)
	clickHouseDateTime64Re     = regexp.MustCompile(`^DateTime64\(\d+\)$`)
	clickHouseEnumRe           = regexp.MustCompile(`^Enum(8|16)\(.+\)$`)
	hiveArrayRe                = regexp.MustCompile(`^array<.+>$`)
	hiveMapRe                  = regexp.MustCompile(`^map<.+>$`)
	hiveStructRe               = regexp.MustCompile(`^struct<.+>$`)
	simpleIdentifierRe         = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	queryTableRe               = regexp.MustCompile(`(?i)(?:from|join|into|table)\s+([a-zA-Z0-9_.` + "`" + `]+)`)
)

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

func extractTableFromQuery(query string) string {
	matches := queryTableRe.FindStringSubmatch(query)
	if len(matches) < 2 {
		return ""
	}
	return strings.Trim(matches[1], "`")
}

func observeConnectorError(connectorType, operation string, err error) {
	if err == nil {
		return
	}
	code := ErrorCodeDriverError
	var connErr *ConnectorError
	if ok := AsConnectorError(err, &connErr); ok {
		code = connErr.Code
	}
	getConnectorMetrics().OperationErrorsTotal.WithLabelValues(connectorType, operation, code).Inc()
}

func observeConnectorOperation(connectorType, operation string, started time.Time, err error) {
	getConnectorMetrics().OperationDuration.WithLabelValues(connectorType, operation).Observe(time.Since(started).Seconds())
	observeConnectorError(connectorType, operation, err)
}

func observeSchemaMetrics(connectorType string, tables []model.DiscoveredTable) {
	getConnectorMetrics().SchemaTablesFound.WithLabelValues(connectorType).Add(float64(len(tables)))
	for _, table := range tables {
		for _, column := range table.Columns {
			if column.InferredPII && column.InferredPIIType != "" {
				getConnectorMetrics().PIIColumnsDetected.WithLabelValues(connectorType, column.InferredPIIType).Inc()
			}
		}
	}
}

func observeFetchMetrics(connectorType string, rows int, approxBytes int64) {
	getConnectorMetrics().FetchRowsTotal.WithLabelValues(connectorType).Add(float64(rows))
	if approxBytes > 0 {
		getConnectorMetrics().FetchBytesTotal.WithLabelValues(connectorType).Add(float64(approxBytes))
	}
}

func clickHouseTypeMapping(native string) (mapped string, subtype string, nullable bool) {
	value := strings.TrimSpace(native)
	for {
		if matches := clickHouseNullableRe.FindStringSubmatch(value); len(matches) == 2 {
			nullable = true
			value = strings.TrimSpace(matches[1])
			continue
		}
		if matches := clickHouseLowCardinalityRe.FindStringSubmatch(value); len(matches) == 2 {
			value = strings.TrimSpace(matches[1])
			continue
		}
		break
	}

	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "uint"), strings.HasPrefix(lower, "int"):
		return "integer", "", nullable
	case strings.HasPrefix(lower, "float"):
		return "float", "", nullable
	case clickHouseDecimalRe.MatchString(value):
		return "decimal", "", nullable
	case lower == "string" || clickHouseFixedStringRe.MatchString(value):
		return "string", "", nullable
	case lower == "uuid":
		return "string", "uuid", nullable
	case lower == "date" || lower == "date32":
		return "datetime", "date", nullable
	case lower == "datetime" || clickHouseDateTime64Re.MatchString(value):
		return "datetime", "", nullable
	case lower == "bool":
		return "boolean", "", nullable
	case clickHouseEnumRe.MatchString(value):
		return "string", "enum", nullable
	case clickHouseArrayRe.MatchString(value):
		return "array", "", nullable
	case strings.HasPrefix(lower, "map("), strings.HasPrefix(lower, "tuple("), lower == "json":
		return "json", "", nullable
	case lower == "ipv4" || lower == "ipv6":
		return "string", "ip", nullable
	case strings.Contains(lower, "point"), strings.Contains(lower, "polygon"), strings.Contains(lower, "ring"):
		return "string", "geometry", nullable
	default:
		mapped := discovery.MapNativeType(value)
		return mapped.Type, mapped.Subtype, nullable
	}
}

func hiveLikeTypeMapping(native string) (mapped string, subtype string) {
	value := strings.ToLower(strings.TrimSpace(native))
	switch {
	case value == "tinyint" || value == "smallint" || value == "int" || value == "bigint":
		return "integer", ""
	case value == "float" || value == "double":
		return "float", ""
	case strings.HasPrefix(value, "decimal"):
		return "decimal", ""
	case value == "string" || strings.HasPrefix(value, "varchar") || strings.HasPrefix(value, "char"):
		return "string", ""
	case value == "timestamp":
		return "datetime", ""
	case value == "date":
		return "datetime", "date"
	case value == "boolean":
		return "boolean", ""
	case hiveArrayRe.MatchString(value):
		return "array", ""
	case hiveMapRe.MatchString(value), hiveStructRe.MatchString(value):
		return "json", ""
	case value == "binary":
		return "binary", ""
	default:
		mapped := discovery.MapNativeType(value)
		return mapped.Type, mapped.Subtype
	}
}

func inputFormatToFormat(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "parquet"):
		return "parquet"
	case strings.Contains(lower, "orc"):
		return "orc"
	case strings.Contains(lower, "avro"):
		return "avro"
	case strings.Contains(lower, "text"):
		return "text"
	case strings.Contains(lower, "json"):
		return "json"
	default:
		return "managed"
	}
}

func isSafeIdentifier(value string) bool {
	return simpleIdentifierRe.MatchString(value)
}

func quoteDotBacktickIdentifier(value string) string {
	parts := strings.Split(value, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, backtickQuote(part))
	}
	return strings.Join(quoted, ".")
}

func parseInt64Loose(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}
