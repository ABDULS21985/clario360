package adapters

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// CSVAdapter parses a CSV file with header row: type,value,severity,confidence,country,tags
type CSVAdapter struct{}

func NewCSVAdapter() *CSVAdapter { return &CSVAdapter{} }

func (a *CSVAdapter) SourceType() string { return "csv_url" }

func (a *CSVAdapter) Parse(_ context.Context, raw []byte) ([]NormalizedIndicator, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1 // variable

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("csv adapter: read header: %w", err)
	}
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	now := time.Now().UTC()
	var out []NormalizedIndicator
	lineNum := 1
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv adapter: line %d: %w", lineNum, err)
		}
		lineNum++

		ind := NormalizedIndicator{
			FirstSeen: now,
			LastSeen:  now,
		}

		if i, ok := colIdx["type"]; ok && i < len(record) {
			ind.IOCType = strings.TrimSpace(record[i])
		}
		if i, ok := colIdx["value"]; ok && i < len(record) {
			ind.IOCValue = strings.TrimSpace(record[i])
		}
		if i, ok := colIdx["severity"]; ok && i < len(record) {
			ind.SeverityCode = strings.TrimSpace(strings.ToLower(record[i]))
		}
		if ind.SeverityCode == "" {
			ind.SeverityCode = "medium"
		}
		if i, ok := colIdx["confidence"]; ok && i < len(record) {
			if c, err := strconv.ParseFloat(strings.TrimSpace(record[i]), 64); err == nil {
				ind.ConfidenceScore = c
			}
		}
		if ind.ConfidenceScore == 0 {
			ind.ConfidenceScore = 0.5
		}
		if i, ok := colIdx["country"]; ok && i < len(record) {
			ind.OriginCountryCode = strings.TrimSpace(record[i])
		}
		if i, ok := colIdx["tags"]; ok && i < len(record) {
			for _, t := range strings.Split(record[i], ";") {
				t = strings.TrimSpace(t)
				if t != "" {
					ind.Tags = append(ind.Tags, t)
				}
			}
		}
		if i, ok := colIdx["title"]; ok && i < len(record) {
			ind.Title = strings.TrimSpace(record[i])
		}
		if ind.Title == "" {
			ind.Title = fmt.Sprintf("Indicator: %s %s", ind.IOCType, ind.IOCValue)
		}

		ind.ExternalRef = fmt.Sprintf("csv-line-%d-%s", lineNum, ind.IOCValue)

		if ind.IOCValue != "" {
			out = append(out, ind)
		}
	}
	return out, nil
}
