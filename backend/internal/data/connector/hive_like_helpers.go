package connector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/beltran/gohive"
)

type hiveFormattedMetadata struct {
	Location         string
	NumRows          int64
	RawDataSize      int64
	InputFormat      string
	PartitionColumns []string
	LastDDLTime      *time.Time
}

func gohiveQueryRows(ctx context.Context, conn *gohive.Connection, query string, limit int) ([][]string, []map[string]any, error) {
	cursor := conn.Cursor()
	defer cursor.Close()
	cursor.Exec(ctx, query)
	if cursor.Err != nil {
		return nil, nil, cursor.Err
	}
	description := cursor.Description()
	rows := make([]map[string]any, 0)
	for cursor.HasMore(ctx) {
		if cursor.Err != nil {
			return description, rows, cursor.Err
		}
		row := cursor.RowMap(ctx)
		if cursor.Err != nil {
			return description, rows, cursor.Err
		}
		rows = append(rows, normalizeHiveRowMap(row))
		if limit > 0 && len(rows) >= limit {
			break
		}
	}
	return description, rows, nil
}

func gohiveExec(ctx context.Context, conn *gohive.Connection, query string) error {
	cursor := conn.Cursor()
	defer cursor.Close()
	cursor.Exec(ctx, query)
	return cursor.Err
}

func normalizeHiveRowMap(row map[string]any) map[string]any {
	if row == nil {
		return nil
	}
	out := make(map[string]any, len(row))
	for key, value := range row {
		out[strings.TrimPrefix(key, "tab_name.")] = value
	}
	return out
}

func firstRowValue(row map[string]any) string {
	for _, value := range row {
		return fmt.Sprint(value)
	}
	return ""
}

func parseHiveDescribeFormatted(rows []map[string]any) hiveFormattedMetadata {
	meta := hiveFormattedMetadata{}
	inPartitionSection := false
	for _, row := range rows {
		colName := strings.TrimSpace(fmt.Sprint(row["col_name"]))
		dataType := strings.TrimSpace(fmt.Sprint(row["data_type"]))
		if colName == "" && dataType == "" {
			continue
		}
		lowerCol := strings.ToLower(colName)
		if strings.HasPrefix(lowerCol, "# partition information") {
			inPartitionSection = true
			continue
		}
		if strings.HasPrefix(lowerCol, "# detailed table information") || strings.HasPrefix(lowerCol, "# storage information") {
			inPartitionSection = false
		}
		if inPartitionSection && !strings.HasPrefix(colName, "#") && colName != "" && dataType != "" {
			meta.PartitionColumns = append(meta.PartitionColumns, colName)
			continue
		}
		switch strings.ToLower(strings.TrimSuffix(colName, ":")) {
		case "location":
			meta.Location = dataType
		case "numrows":
			meta.NumRows = parseInt64Loose(dataType)
		case "rawdatasize":
			meta.RawDataSize = parseInt64Loose(dataType)
		case "inputformat":
			meta.InputFormat = dataType
		case "transient_lastddltime":
			if ts := parseInt64Loose(dataType); ts > 0 {
				value := time.Unix(ts, 0).UTC()
				meta.LastDDLTime = &value
			}
		}
	}
	return meta
}
