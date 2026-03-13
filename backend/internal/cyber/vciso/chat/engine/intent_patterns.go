package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

// ---------------------------------------------------------------------------
// intentSpec — declarative pattern definition
// ---------------------------------------------------------------------------

// intentSpec is the human-authored definition of an intent.  It is compiled
// into a chatmodel.IntentPattern at registration time.  Using a separate
// struct keeps the declaration site clean (no regexp.MustCompile inline).
type intentSpec struct {
	Intent         string
	ToolName       string
	PatternTexts   []string
	Keywords       []string
	RequiresEntity bool
	EntityType     string
	Priority       int
	Description    string
}

// ---------------------------------------------------------------------------
// PatternRegistry — thread-safe, validating intent store
// ---------------------------------------------------------------------------

// PatternRegistry owns the full set of compiled intent patterns.  It is
// the single source of truth for the IntentClassifier.
//
// Design goals:
//   - Compile-time validation: bad regex → clear error, not a runtime panic
//   - Duplicate detection: same intent registered twice → warning + merge
//   - Runtime extensibility: plugins can register patterns after init
//   - Priority-sorted access: highest-priority intents evaluated first
//   - Thread-safe reads after init (writes are rare, behind a mutex)
type PatternRegistry struct {
	mu       sync.RWMutex
	patterns []*chatmodel.IntentPattern
	index    map[string]int // intent name → position in patterns slice
	logger   zerolog.Logger
}

// NewPatternRegistry creates an empty registry.
func NewPatternRegistry(logger ...zerolog.Logger) *PatternRegistry {
	l := zerolog.Nop()
	if len(logger) > 0 {
		l = logger[0]
	}
	return &PatternRegistry{
		index:  make(map[string]int),
		logger: l,
	}
}

// NewDefaultPatternRegistry creates a registry pre-loaded with the built-in
// vCISO intent patterns.
func NewDefaultPatternRegistry(logger ...zerolog.Logger) *PatternRegistry {
	reg := NewPatternRegistry(logger...)
	for _, spec := range builtinIntentSpecs() {
		if err := reg.Register(spec); err != nil {
			// Built-in specs should never fail.  If they do, it's a code bug.
			panic(fmt.Sprintf("intent_patterns: built-in spec %q failed: %v", spec.Intent, err))
		}
	}
	return reg
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// Register compiles and adds a single intent spec to the registry.
// Returns an error if any regex fails to compile.
//
// If an intent with the same name already exists, the patterns and keywords
// are merged (not replaced) and the higher priority wins.  This lets plugins
// extend built-in intents without losing the base patterns.
func (r *PatternRegistry) Register(spec intentSpec) error {
	compiled, err := compileSpec(spec)
	if err != nil {
		return fmt.Errorf("intent %q: %w", spec.Intent, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if idx, exists := r.index[spec.Intent]; exists {
		r.mergeAt(idx, compiled, spec)
		r.logger.Debug().
			Str("intent", spec.Intent).
			Msg("merged into existing intent pattern")
	} else {
		r.index[spec.Intent] = len(r.patterns)
		r.patterns = append(r.patterns, compiled)
	}

	r.sortLocked()
	return nil
}

// RegisterAll registers multiple specs, stopping at the first error.
func (r *PatternRegistry) RegisterAll(specs []intentSpec) error {
	for _, spec := range specs {
		if err := r.Register(spec); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Access
// ---------------------------------------------------------------------------

// All returns a snapshot of all registered patterns, sorted by priority
// descending.  The returned slice is a copy — safe to iterate without
// holding a lock.
func (r *PatternRegistry) All() []*chatmodel.IntentPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*chatmodel.IntentPattern, len(r.patterns))
	copy(out, r.patterns)
	return out
}

// Get returns the pattern for a specific intent, or nil if not found.
func (r *PatternRegistry) Get(intent string) *chatmodel.IntentPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if idx, ok := r.index[intent]; ok {
		return r.patterns[idx]
	}
	return nil
}

// Len returns the number of registered intents.
func (r *PatternRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.patterns)
}

// IntentNames returns all registered intent names in priority order.
func (r *PatternRegistry) IntentNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.patterns))
	for i, p := range r.patterns {
		names[i] = p.Intent
	}
	return names
}

// ---------------------------------------------------------------------------
// Removal (for testing / hot-reload)
// ---------------------------------------------------------------------------

// Remove deletes an intent by name.  Returns true if it existed.
func (r *PatternRegistry) Remove(intent string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	idx, ok := r.index[intent]
	if !ok {
		return false
	}

	// Swap-delete from the slice.
	last := len(r.patterns) - 1
	if idx != last {
		r.patterns[idx] = r.patterns[last]
		r.index[r.patterns[idx].Intent] = idx
	}
	r.patterns = r.patterns[:last]
	delete(r.index, intent)

	r.sortLocked()
	return true
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// Validate checks the registry for common issues:
//   - intents with zero patterns AND zero keywords
//   - duplicate tool names across different intents
//   - pattern strings that fail to compile (should not happen if Register
//     was used, but useful after hot-reload)
//
// Returns a list of human-readable warnings.
func (r *PatternRegistry) Validate() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var warnings []string
	toolUsers := make(map[string][]string) // tool → []intent

	for _, p := range r.patterns {
		if len(p.Patterns) == 0 && len(p.Keywords) == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"intent %q has no patterns and no keywords — it can never match", p.Intent))
		}
		toolUsers[p.ToolName] = append(toolUsers[p.ToolName], p.Intent)
	}

	for tool, intents := range toolUsers {
		if len(intents) > 1 {
			warnings = append(warnings, fmt.Sprintf(
				"tool %q is used by multiple intents: %s", tool, strings.Join(intents, ", ")))
		}
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

func (r *PatternRegistry) mergeAt(idx int, incoming *chatmodel.IntentPattern, spec intentSpec) {
	existing := r.patterns[idx]

	// Merge patterns (append, skip exact duplicates).
	existingTexts := make(map[string]struct{}, len(existing.PatternStrings))
	for _, t := range existing.PatternStrings {
		existingTexts[t] = struct{}{}
	}
	for i, text := range spec.PatternTexts {
		if _, dup := existingTexts[text]; dup {
			continue
		}
		existing.Patterns = append(existing.Patterns, incoming.Patterns[i])
		existing.PatternStrings = append(existing.PatternStrings, text)
	}

	// Merge keywords (append, skip duplicates).
	existingKW := make(map[string]struct{}, len(existing.Keywords))
	for _, k := range existing.Keywords {
		existingKW[strings.ToLower(k)] = struct{}{}
	}
	for _, k := range spec.Keywords {
		if _, dup := existingKW[strings.ToLower(k)]; dup {
			continue
		}
		existing.Keywords = append(existing.Keywords, k)
	}

	// Take the higher priority.
	if spec.Priority > existing.Priority {
		existing.Priority = spec.Priority
	}

	// Prefer non-empty description.
	if spec.Description != "" {
		existing.Description = spec.Description
	}

	// Merge entity requirements.
	if spec.RequiresEntity {
		existing.RequiresEntity = true
		if spec.EntityType != "" {
			existing.EntityType = spec.EntityType
		}
	}
}

func (r *PatternRegistry) sortLocked() {
	sort.SliceStable(r.patterns, func(i, j int) bool {
		if r.patterns[i].Priority != r.patterns[j].Priority {
			return r.patterns[i].Priority > r.patterns[j].Priority
		}
		// Stable tie-breaking by name (avoids fmt.Sprintf allocation).
		return r.patterns[i].Intent < r.patterns[j].Intent
	})

	// Rebuild index after sort.
	for i, p := range r.patterns {
		r.index[p.Intent] = i
	}
}

// ---------------------------------------------------------------------------
// Compilation
// ---------------------------------------------------------------------------

// compileSpec transforms a declarative intentSpec into a compiled IntentPattern.
// Unlike regexp.MustCompile, it returns an error instead of panicking.
func compileSpec(spec intentSpec) (*chatmodel.IntentPattern, error) {
	patterns := make([]*regexp.Regexp, 0, len(spec.PatternTexts))
	for _, text := range spec.PatternTexts {
		compiled, err := regexp.Compile(text)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", text, err)
		}
		patterns = append(patterns, compiled)
	}

	return &chatmodel.IntentPattern{
		Intent:         spec.Intent,
		ToolName:       spec.ToolName,
		Patterns:       patterns,
		PatternStrings: append([]string(nil), spec.PatternTexts...),
		Keywords:       append([]string(nil), spec.Keywords...),
		RequiresEntity: spec.RequiresEntity,
		EntityType:     spec.EntityType,
		Priority:       spec.Priority,
		Description:    spec.Description,
	}, nil
}

// ---------------------------------------------------------------------------
// defaultIntentPatterns — convenience wrapper for the classifier
// ---------------------------------------------------------------------------

// defaultIntentPatterns returns the compiled built-in patterns sorted by
// priority.  This is the bridge between the registry and the
// IntentClassifier's existing constructor.
func defaultIntentPatterns() []*chatmodel.IntentPattern {
	return NewDefaultPatternRegistry().All()
}

// ---------------------------------------------------------------------------
// Built-in intent specifications
// ---------------------------------------------------------------------------

func builtinIntentSpecs() []intentSpec {
	alertIDLike := `(?:#?\d+|[0-9a-f]{8}(?:-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})?)`

	return []intentSpec{
		{
			Intent:      "risk_score_query",
			ToolName:    "risk_score",
			Priority:    80,
			Description: "Check the organization's current security risk score and grade",
			PatternTexts: []string{
				`(?i)(what|show|tell|get).*(risk|security)\s*(score|posture|status|level|rating)`,
				`(?i)how\s*(secure|safe|risky|vulnerable)\s*(are we|is our|is the)`,
				`(?i)(risk|security)\s*(summary|assessment)`,
				`(?i)(current|overall|org|our).*(risk|security).*(score|status|level)`,
				`(?i)(what'?s|whats)\s*(the|our)\s*(risk|security)`,
			},
			Keywords: []string{"risk", "overview", "score", "posture", "security", "status"},
		},
		{
			Intent:      "alert_query",
			ToolName:    "alert_summary",
			Priority:    75,
			Description: "View alert counts and recent critical/high alerts",
			PatternTexts: []string{
				`(?i)(how many|count|show|list|get|display).*(alert|incident|threat|warning)s?`,
				`(?i)(critical|high|open|unresolved|new|active|pending).*(alert|incident)s?`,
				`(?i)alert.*(summary|overview|status|count|breakdown|this week|recent)`,
				`(?i)(any|are there).*(new|open|unresolved).*(alert|incident)s?`,
				`(?i)(alert|incident|threat).*(volume|number|total)`,
			},
			Keywords: []string{"alerts", "today", "critical", "open", "incidents", "threats"},
		},
		{
			Intent:         "alert_detail",
			ToolName:       "alert_detail",
			Priority:       85,
			Description:    "Get detailed information about a specific security alert",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts: []string{
				`(?i)(show|tell|get|details?|info|about).*(alert|incident)\s*` + alertIDLike,
				`(?i)(what happened|explain|describe).*(alert|incident)\s*` + alertIDLike,
				`(?i)alert\s*` + alertIDLike,
				`(?i)\binvestigat(e|ion)\b.*\b(alert|incident)\b.*\b(details?|info|about)\b`,
				`(?i)(show|tell|get).*(details?|info|about).*(first|second|third|last|it|that|this)$`,
			},
			Keywords: []string{"alert details", "about alert", "what happened", "explain alert", "alert info"},
		},
		{
			Intent:         "asset_lookup",
			ToolName:       "asset_lookup",
			Priority:       70,
			Description:    "Look up details about a specific asset, server, or device",
			RequiresEntity: true,
			EntityType:     "asset_name",
			PatternTexts: []string{
				`(?i)(show|tell|get|find|look\s*up).*(about|detail|info).*(asset|server|host|machine|device|endpoint)`,
				`(?i)(what do we know about|show details?\s+for|details?\s*(for|on|about))\s+(?:\d{1,3}(?:\.\d{1,3}){3}|[\w.-]+)`,
				`(?i)(asset|server|host|device)\s+[\w.-]+\s*(info|detail|status|vuln)`,
				`(?i)(show|tell|get|find|look\s*up).*(web|db|host|srv|prod|dev|corp)[\w.-]*`,
			},
			Keywords: []string{"asset", "server", "host", "device", "machine", "details about", "look up"},
		},
		{
			Intent:      "vulnerability_query",
			ToolName:    "vulnerability_summary",
			Priority:    70,
			Description: "View top vulnerabilities across the organization",
			PatternTexts: []string{
				`(?i)(top|worst|critical|unpatched|open).*(vuln|cve|weakness|exposure)s?`,
				`(?i)(vuln|cve|weakness).*(summary|overview|top|critical|list|count|worst)`,
				`(?i)(show|list|what are).*(vuln|cve)s?`,
			},
			Keywords: []string{"vulnerabilities", "CVEs", "unpatched", "weaknesses", "exposures", "top vulns"},
		},
		{
			Intent:      "vulnerability_priority_query",
			ToolName:    "get_vulnerability_priority",
			Priority:    84,
			Description: "Predict which vulnerabilities are most likely to be exploited next",
			PatternTexts: []string{
				`(?i)(which|what).*(cve|vuln|vulnerability).*(priorit|patch first|exploit|most likely)`,
				`(?i)(patch|prioritiz).*(open|our).*(cve|vuln)`,
				`(?i)(exploit probability|predicted exploit|most urgent cves?)`,
			},
			Keywords: []string{"patch first", "exploit probability", "prioritize vulnerabilities", "predicted exploit", "open cves"},
		},
		{
			Intent:      "mitre_query",
			ToolName:    "mitre_coverage",
			Priority:    75,
			Description: "Check MITRE ATT&CK detection coverage and gaps",
			PatternTexts: []string{
				`(?i)mitre.*(coverage|matrix|att&?ck|detection|gap)`,
				`(?i)(detection|att&?ck).*(coverage|gap|matrix)`,
				`(?i)(what|which).*(technique|tactic).*(cover|detect|miss)`,
			},
			Keywords: []string{"MITRE", "ATT&CK", "coverage", "detection gaps", "techniques", "tactics"},
		},
		{
			Intent:      "ueba_query",
			ToolName:    "ueba_summary",
			Priority:    70,
			Description: "View users and entities with anomalous behavioral patterns",
			PatternTexts: []string{
				`(?i)(risky|suspicious|anomal|unusual).*(user|entity|account|employee|person)s?`,
				`(?i)(who|which).*(riskiest|risky|suspicious|anomal|unusual).*(user|entity|person|account)`,
				`(?i)(insider|behavioral).*(threat|risk|anomal)`,
				`(?i)ueba.*(summary|alert|risk|top)`,
				`(?i)(suspicious|anomalous)\s+activity`,
			},
			Keywords: []string{"risky users", "suspicious activity", "behavioral", "UEBA", "insider threat", "anomalous"},
		},
		{
			Intent:      "insider_forecast_query",
			ToolName:    "get_insider_threat_forecast",
			Priority:    82,
			Description: "Forecast which users are trending toward insider threat thresholds",
			PatternTexts: []string{
				`(?i)(which|who).*(user|account).*(trending|trajectory|forecast|projected).*(insider|risk)`,
				`(?i)(insider|behavioral).*(forecast|trajectory|next 7|next 30)`,
				`(?i)(who).*(risk score).*(projected|escalate|increase)`,
			},
			Keywords: []string{"insider forecast", "risk trajectory", "projected risk", "escalating users", "behavioral forecast"},
		},
		{
			Intent:      "pipeline_query",
			ToolName:    "pipeline_status",
			Priority:    65,
			Description: "Check data pipeline health and failures",
			PatternTexts: []string{
				`(?i)(pipeline|etl|data flow|data job).*(status|fail|error|running|broken|down|health)`,
				`(?i)(any|are).*(pipeline|etl|data).*(fail|down|broken|error|stuck)`,
				`(?i)(data|pipeline).*(health|status|problem)`,
			},
			Keywords: []string{"pipeline", "ETL", "data flow", "failing", "broken", "data job", "data health"},
		},
		{
			Intent:      "compliance_query",
			ToolName:    "compliance_score",
			Priority:    70,
			Description: "Check compliance status across regulatory frameworks",
			PatternTexts: []string{
				`(?i)(compliance|iso|nca|sama|nist|soc2|gdpr|hipaa|pci).*(score|status|gap|ready|progress|summary)`,
				`(?i)(how|are we).*(compliant|compliance|audit.ready)`,
				`(?i)(audit|regulatory|framework).*(status|readiness|score)`,
			},
			Keywords: []string{"compliance", "ISO 27001", "NCA", "SAMA", "NIST", "SOC2", "audit ready", "framework"},
		},
		{
			Intent:      "recommendation_query",
			ToolName:    "recommendation",
			Priority:    80,
			Description: "Get personalized recommendations on what to focus on",
			PatternTexts: []string{
				`(?i)\b(what should|recommend|suggest|prioriti[sz]e|focus|action|todo)\b`,
				`(?i)\b(top|most important|urgent|critical)\b.*\b(action|task|priorit(?:y|ies)|thing|issue)s?\b`,
				`(?i)\b(what|where)\b.*\b(should|do|focus|start)\b.*\b(first|today|now|next)\b`,
				`(?i)\b(daily|morning|today)\b.*\b(brief|summary|priority|action)\b`,
				`(?i)\bwhat\b.*\b(matters?|urgent|important|press)\b.*\b(today|now|right now)\b`,
			},
			Keywords: []string{"recommend", "suggest", "prioritize", "focus", "should I", "action items", "what next", "today"},
		},
		{
			Intent:         "dashboard_build",
			ToolName:       "dashboard_builder",
			Priority:       82,
			Description:    "Build a custom dashboard with specified metrics and charts",
			RequiresEntity: true,
			EntityType:     "description",
			PatternTexts: []string{
				`(?i)(build|create|make|generate|set up).*(dashboard|view|board|panel)`,
				`(?i)(dashboard|view|panel).*(for|about|showing|with|track)`,
				`(?i)(i need|give me|can you make).*(dashboard|view|chart|visual)`,
			},
			Keywords: []string{"build dashboard", "create view", "custom dashboard", "make dashboard", "visualization"},
		},
		{
			Intent:         "investigation_query",
			ToolName:       "investigation",
			Priority:       90,
			Description:    "Run a comprehensive investigation on a specific alert",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts: []string{
				`(?i)investigat.*(alert|incident|event|threat)\s*` + alertIDLike,
				`(?i)(deep dive|analy[sz]e|look into|dig into|examine).*(alert|incident)\s*` + alertIDLike,
				`(?i)(full|complete|detailed).*(investigation|analysis).*(alert|incident)\s*` + alertIDLike,
				`(?i)investigat(e|ion).*(it|that|this|first|second|third|last)`,
			},
			Keywords: []string{"investigate", "deep dive", "analyze", "look into", "examine", "dig into"},
		},
		{
			Intent:      "trend_query",
			ToolName:    "trend_analysis",
			Priority:    76,
			Description: "Analyze security trends and how metrics have changed",
			PatternTexts: []string{
				`(?i)(how|has).*(risk|alert|score|posture|threat).*(chang|trend|evolv|compar|improv|worsen)`,
				`(?i)(trend|change|history|comparison|progress).*(risk|alert|score|threat|security)`,
				`(?i)(alert|risk|threat).*(volume|count|score).*(trend|change|history)`,
				`(?i)(over time|week over week|month over month|getting better|getting worse)`,
				`(?i)(compar).*(this week|last week|this month|last month|yesterday)`,
			},
			Keywords: []string{"trend", "change", "history", "comparison", "over time", "progress", "getting better"},
		},
		{
			Intent:      "threat_forecast_query",
			ToolName:    "get_threat_forecast",
			Priority:    83,
			Description: "Forecast alert volume, attack technique shifts, and campaign activity",
			PatternTexts: []string{
				`(?i)(how many|forecast|predict).*(alert|alerts).*(week|month|days)`,
				`(?i)(emerging|increasing).*(attack technique|technique|phishing|campaign)`,
				`(?i)(coordinated attack|campaign).*(predict|forecast|detect)`,
				`(?i)(threat|attack).*(forecast|trend|next 7|next 30|next 90)`,
			},
			Keywords: []string{"threat forecast", "alert forecast", "emerging techniques", "campaign detection", "next 30 days"},
		},
		{
			Intent:      "asset_risk_prediction_query",
			ToolName:    "get_asset_risk_prediction",
			Priority:    85,
			Description: "Predict which assets are most likely to be targeted next",
			PatternTexts: []string{
				`(?i)(which|what).*(asset|server|endpoint|database).*(most at risk|most likely targeted|targeted next)`,
				`(?i)(predict|forecast).*(asset|server|endpoint).*(target|attack)`,
				`(?i)(which servers?).*(patch first|at risk)`,
			},
			Keywords: []string{"most at risk assets", "targeted next", "at risk servers", "asset risk prediction", "patch first servers"},
		},
		{
			Intent:         "remediation_query",
			ToolName:       "remediation",
			Priority:       86,
			Description:    "Start a governed remediation action for an alert or vulnerability",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts: []string{
				`(?i)(start|run|execute|trigger|initiate|begin).*(remediat|fix|patch|mitiga|resolv).*(alert|vuln|issue|threat|incident)?\s*` + alertIDLike + `?`,
				`(?i)remediat.*(alert|vuln|issue|threat|incident)\s*` + alertIDLike,
				`(?i)(fix|patch|block|isolate|contain).*(alert|threat|vuln)\s*` + alertIDLike,
			},
			Keywords: []string{"remediate", "fix", "patch", "mitigate", "start remediation", "block", "isolate"},
		},
		{
			Intent:      "report_query",
			ToolName:    "report_generator",
			Priority:    70,
			Description: "Generate an executive security report",
			PatternTexts: []string{
				`(?i)(generate|create|produce|write|prepare).*(report|brief|summary|executive|pdf|document)`,
				`(?i)(executive|security|weekly|monthly|quarterly|annual).*(report|brief|summary)`,
				`(?i)(need|want|give me).*(report|brief|summary)`,
			},
			Keywords: []string{"generate report", "executive summary", "security briefing", "weekly report", "pdf"},
		},
	}
}
