package engine

import "regexp"

type timePattern struct {
	Name    string
	Pattern *regexp.Regexp
}

func defaultTimePatterns() []timePattern {
	return []timePattern{
		{Name: "today", Pattern: regexp.MustCompile(`(?i)\btoday\b`)},
		{Name: "yesterday", Pattern: regexp.MustCompile(`(?i)\byesterday\b`)},
		{Name: "this_week", Pattern: regexp.MustCompile(`(?i)\bthis week\b`)},
		{Name: "last_week", Pattern: regexp.MustCompile(`(?i)\blast week\b`)},
		{Name: "last_7_days", Pattern: regexp.MustCompile(`(?i)\b(past week|last 7 days|last seven days|past 7 days)\b`)},
		{Name: "this_month", Pattern: regexp.MustCompile(`(?i)\bthis month\b`)},
		{Name: "last_month", Pattern: regexp.MustCompile(`(?i)\blast month\b`)},
		{Name: "last_30_days", Pattern: regexp.MustCompile(`(?i)\b(last 30 days|past 30 days|past month)\b`)},
		{Name: "last_24_hours", Pattern: regexp.MustCompile(`(?i)\b(last 24 hours|past day)\b`)},
		{Name: "last_hour", Pattern: regexp.MustCompile(`(?i)\blast hour\b`)},
	}
}

