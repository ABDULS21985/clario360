package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// TimeRange — the resolved output of a time pattern match
// ---------------------------------------------------------------------------

// TimeRange represents a concrete start/end window resolved from a
// natural-language time reference.
type TimeRange struct {
	Name    string    // canonical name (e.g. "last_7_days")
	Start   time.Time // inclusive
	End     time.Time // exclusive
	Pattern string    // the regex text that matched (for debugging)
}

// Duration returns the span of the time range.
func (tr TimeRange) Duration() time.Duration { return tr.End.Sub(tr.Start) }

// Contains returns true if t falls within [Start, End).
func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && t.Before(tr.End)
}

// ---------------------------------------------------------------------------
// timePattern — a single pattern definition
// ---------------------------------------------------------------------------

// timePattern pairs a canonical name with a compiled regex and a resolver
// function that computes the concrete time range relative to "now".
type timePattern struct {
	Name     string
	Pattern  *regexp.Regexp
	Text     string // original regex text (for debugging / serialisation)
	Priority int    // higher = matched first when multiple patterns fire
	Resolve  func(now time.Time) TimeRange
}

// ---------------------------------------------------------------------------
// TimeResolver — the public API
// ---------------------------------------------------------------------------

// TimeResolver extracts and resolves natural-language time references from
// user messages into concrete TimeRange values.
//
// It is stateless and safe for concurrent use after construction.
type TimeResolver struct {
	patterns []timePattern
	now      func() time.Time
}

// TimeResolverOption configures the resolver.
type TimeResolverOption func(*TimeResolver)

// WithTimeNow injects a clock function (useful for deterministic tests).
func WithTimeNow(fn func() time.Time) TimeResolverOption {
	return func(r *TimeResolver) {
		if fn != nil {
			r.now = fn
		}
	}
}

// WithExtraTimePatterns appends additional patterns after the defaults.
func WithExtraTimePatterns(patterns ...timePattern) TimeResolverOption {
	return func(r *TimeResolver) {
		r.patterns = append(r.patterns, patterns...)
	}
}

// WithTimePatterns replaces the entire pattern set.
func WithTimePatterns(patterns []timePattern) TimeResolverOption {
	return func(r *TimeResolver) {
		if len(patterns) > 0 {
			r.patterns = patterns
		}
	}
}

// NewTimeResolver creates a resolver with the default patterns and any
// supplied options.
func NewTimeResolver(opts ...TimeResolverOption) *TimeResolver {
	r := &TimeResolver{
		patterns: defaultTimePatterns(),
		now:      func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(r)
	}

	// Sort by priority descending so higher-priority patterns match first.
	sort.SliceStable(r.patterns, func(i, j int) bool {
		return r.patterns[i].Priority > r.patterns[j].Priority
	})

	return r
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Resolve scans the message for a time reference and returns the first
// (highest-priority) match as a concrete TimeRange.  Returns nil if no
// pattern matches.
func (r *TimeResolver) Resolve(message string) *TimeRange {
	normalised := strings.ToLower(strings.TrimSpace(message))
	if normalised == "" {
		return nil
	}

	now := r.now()
	for _, tp := range r.patterns {
		if tp.Pattern.MatchString(normalised) {
			tr := r.resolveWithMessage(tp, normalised, now)
			return &tr
		}
	}

	return nil
}

// ResolveAll returns every matching time pattern in the message, sorted
// by priority descending.  Useful for messages like "compare last week
// and this month" that reference multiple periods.
func (r *TimeResolver) ResolveAll(message string) []TimeRange {
	normalised := strings.ToLower(strings.TrimSpace(message))
	if normalised == "" {
		return nil
	}

	now := r.now()
	var results []TimeRange

	for _, tp := range r.patterns {
		if tp.Pattern.MatchString(normalised) {
			tr := r.resolveWithMessage(tp, normalised, now)
			results = append(results, tr)
		}
	}

	return results
}

// ResolveName returns the canonical name of the first matching time
// pattern, or "" if none match.  This is the lightweight path when you
// only need the name for filter carry-over.
func (r *TimeResolver) ResolveName(message string) string {
	tr := r.Resolve(message)
	if tr == nil {
		return ""
	}
	return tr.Name
}

// PatternNames returns all registered pattern names in priority order.
func (r *TimeResolver) PatternNames() []string {
	names := make([]string, len(r.patterns))
	for i, p := range r.patterns {
		names[i] = p.Name
	}
	return names
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// Validate checks for duplicate pattern names.
func (r *TimeResolver) Validate() []string {
	var warnings []string
	seen := make(map[string]int)

	for i, p := range r.patterns {
		if prev, dup := seen[p.Name]; dup {
			warnings = append(warnings, fmt.Sprintf(
				"duplicate pattern name %q at indices %d and %d", p.Name, prev, i))
		}
		seen[p.Name] = i
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Message-aware resolution
// ---------------------------------------------------------------------------

// resolveWithMessage handles "last N <unit>" patterns by extracting N from
// the message.  For all other patterns it delegates to tp.Resolve(now).
func (r *TimeResolver) resolveWithMessage(tp timePattern, message string, now time.Time) TimeRange {
	if strings.HasPrefix(tp.Name, "last_n_") {
		n := extractNumber(tp.Pattern, message)
		if n > 0 {
			var unit time.Duration
			switch {
			case strings.Contains(tp.Name, "hours"):
				unit = time.Hour
			case strings.Contains(tp.Name, "days"):
				unit = 24 * time.Hour
			case strings.Contains(tp.Name, "weeks"):
				unit = 7 * 24 * time.Hour
			}
			if unit > 0 {
				return TimeRange{
					Name:    tp.Name,
					Start:   now.Add(-time.Duration(n) * unit),
					End:     now,
					Pattern: tp.Text,
				}
			}
		}
	}

	tr := tp.Resolve(now)
	tr.Pattern = tp.Text
	return tr
}

// extractNumber pulls the first captured numeric group from a regex match.
func extractNumber(re *regexp.Regexp, message string) int {
	matches := re.FindStringSubmatch(message)
	if len(matches) < 2 {
		return 0
	}
	// The number is typically in the last capture group.
	numStr := matches[len(matches)-1]
	var n int
	fmt.Sscanf(numStr, "%d", &n)
	return n
}

// ---------------------------------------------------------------------------
// Default time patterns
// ---------------------------------------------------------------------------

func defaultTimePatterns() []timePattern {
	return []timePattern{
		// --- Specific recent windows (highest priority) ---
		{
			Name: "last_hour", Text: `(?i)\blast hour\b`,
			Pattern: regexp.MustCompile(`(?i)\blast hour\b`), Priority: 90,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_hour", Start: now.Add(-time.Hour), End: now}
			},
		},
		{
			Name: "last_24_hours", Text: `(?i)\b(last 24 hours|past 24 hours|past day)\b`,
			Pattern: regexp.MustCompile(`(?i)\b(last 24 hours|past 24 hours|past day)\b`), Priority: 85,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_24_hours", Start: now.Add(-24 * time.Hour), End: now}
			},
		},
		// --- Relative N-unit patterns ---
		{
			Name: "last_n_hours", Text: `(?i)\b(last|past)\s+(\d{1,3})\s+hours?\b`,
			Pattern: regexp.MustCompile(`(?i)\b(last|past)\s+(\d{1,3})\s+hours?\b`), Priority: 88,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_n_hours", Start: now.Add(-time.Hour), End: now}
			},
		},
		{
			Name: "last_n_days", Text: `(?i)\b(last|past)\s+(\d{1,3})\s+days?\b`,
			Pattern: regexp.MustCompile(`(?i)\b(last|past)\s+(\d{1,3})\s+days?\b`), Priority: 68,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_n_days", Start: now.Add(-24 * time.Hour), End: now}
			},
		},
		{
			Name: "last_n_weeks", Text: `(?i)\b(last|past)\s+(\d{1,2})\s+weeks?\b`,
			Pattern: regexp.MustCompile(`(?i)\b(last|past)\s+(\d{1,2})\s+weeks?\b`), Priority: 58,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_n_weeks", Start: now.Add(-7 * 24 * time.Hour), End: now}
			},
		},
		// --- Named days ---
		{
			Name: "today", Text: `(?i)\btoday\b`,
			Pattern: regexp.MustCompile(`(?i)\btoday\b`), Priority: 80,
			Resolve: func(now time.Time) TimeRange {
				sod := startOfDay(now)
				return TimeRange{Name: "today", Start: sod, End: sod.Add(24 * time.Hour)}
			},
		},
		{
			Name: "yesterday", Text: `(?i)\byesterday\b`,
			Pattern: regexp.MustCompile(`(?i)\byesterday\b`), Priority: 80,
			Resolve: func(now time.Time) TimeRange {
				sod := startOfDay(now).Add(-24 * time.Hour)
				return TimeRange{Name: "yesterday", Start: sod, End: sod.Add(24 * time.Hour)}
			},
		},
		// --- Named weeks ---
		{
			Name: "this_week", Text: `(?i)\bthis week\b`,
			Pattern: regexp.MustCompile(`(?i)\bthis week\b`), Priority: 70,
			Resolve: func(now time.Time) TimeRange {
				sow := startOfWeek(now)
				return TimeRange{Name: "this_week", Start: sow, End: now}
			},
		},
		{
			Name: "last_week", Text: `(?i)\blast week\b`,
			Pattern: regexp.MustCompile(`(?i)\blast week\b`), Priority: 70,
			Resolve: func(now time.Time) TimeRange {
				sow := startOfWeek(now)
				return TimeRange{Name: "last_week", Start: sow.Add(-7 * 24 * time.Hour), End: sow}
			},
		},
		{
			Name: "last_7_days", Text: `(?i)\b(past week|last 7 days|last seven days|past 7 days)\b`,
			Pattern: regexp.MustCompile(`(?i)\b(past week|last 7 days|last seven days|past 7 days)\b`), Priority: 65,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_7_days", Start: now.Add(-7 * 24 * time.Hour), End: now}
			},
		},
		// --- Named months ---
		{
			Name: "this_month", Text: `(?i)\bthis month\b`,
			Pattern: regexp.MustCompile(`(?i)\bthis month\b`), Priority: 60,
			Resolve: func(now time.Time) TimeRange {
				som := startOfMonth(now)
				return TimeRange{Name: "this_month", Start: som, End: now}
			},
		},
		{
			Name: "last_month", Text: `(?i)\blast month\b`,
			Pattern: regexp.MustCompile(`(?i)\blast month\b`), Priority: 60,
			Resolve: func(now time.Time) TimeRange {
				som := startOfMonth(now)
				prevSom := startOfMonth(som.Add(-24 * time.Hour))
				return TimeRange{Name: "last_month", Start: prevSom, End: som}
			},
		},
		{
			Name: "last_30_days", Text: `(?i)\b(last 30 days|past 30 days|past month)\b`,
			Pattern: regexp.MustCompile(`(?i)\b(last 30 days|past 30 days|past month)\b`), Priority: 55,
			Resolve: func(now time.Time) TimeRange {
				return TimeRange{Name: "last_30_days", Start: now.Add(-30 * 24 * time.Hour), End: now}
			},
		},
		// --- Quarters ---
		{
			Name: "this_quarter", Text: `(?i)\bthis quarter\b`,
			Pattern: regexp.MustCompile(`(?i)\bthis quarter\b`), Priority: 50,
			Resolve: func(now time.Time) TimeRange {
				soq := startOfQuarter(now)
				return TimeRange{Name: "this_quarter", Start: soq, End: now}
			},
		},
		{
			Name: "last_quarter", Text: `(?i)\blast quarter\b`,
			Pattern: regexp.MustCompile(`(?i)\blast quarter\b`), Priority: 50,
			Resolve: func(now time.Time) TimeRange {
				soq := startOfQuarter(now)
				prevSoq := startOfQuarter(soq.Add(-24 * time.Hour))
				return TimeRange{Name: "last_quarter", Start: prevSoq, End: soq}
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Calendar helpers
// ---------------------------------------------------------------------------

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	// ISO week: Monday = start of week.
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7
	}
	return startOfDay(t.AddDate(0, 0, -(weekday - 1)))
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func startOfQuarter(t time.Time) time.Time {
	month := t.Month()
	quarterStart := month - (month-1)%3
	return time.Date(t.Year(), quarterStart, 1, 0, 0, 0, 0, t.Location())
}
