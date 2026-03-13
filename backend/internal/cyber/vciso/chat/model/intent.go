package model

import "regexp"

type IntentPattern struct {
	Intent         string
	ToolName       string
	Patterns       []*regexp.Regexp
	PatternStrings []string
	Keywords       []string
	RequiresEntity bool
	EntityType     string
	Priority       int
	Description    string
}

type ClassificationResult struct {
	Intent      string            `json:"intent"`
	ToolName    string            `json:"tool_name"`
	Confidence  float64           `json:"confidence"`
	MatchMethod string            `json:"match_method"`
	MatchedRule string            `json:"matched_rule"`
	Entities    map[string]string `json:"entities"`
}
