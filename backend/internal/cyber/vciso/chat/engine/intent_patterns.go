package engine

import (
	"fmt"
	"regexp"
	"sort"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

func defaultIntentPatterns() []*chatmodel.IntentPattern {
	return compileIntentPatterns([]intentSpec{
		{
			Intent:       "risk_score_query",
			ToolName:     "risk_score",
			Priority:     80,
			Description:  "Check the organization's current security risk score and grade",
			PatternTexts: []string{`(?i)(what|show|tell|get).*(risk|security)\s*(score|posture|status|level|rating)`, `(?i)how\s*(secure|safe|risky|vulnerable)\s*(are we|is our|is the)`, `(?i)(risk|security)\s*(overview|summary|dashboard|assessment)`, `(?i)(current|overall|org|our).*(risk|security).*(score|status|level)`, `(?i)(what'?s|whats)\s*(the|our)\s*(risk|security)`},
			Keywords:     []string{"risk", "score", "posture", "security level", "how safe", "security status"},
		},
		{
			Intent:       "alert_query",
			ToolName:     "alert_summary",
			Priority:     75,
			Description:  "View alert counts and recent critical/high alerts",
			PatternTexts: []string{`(?i)(how many|count|show|list|get|display).*(alert|incident|threat|warning)s?`, `(?i)(critical|high|open|unresolved|new|active|pending).*(alert|incident)s?`, `(?i)alert.*(summary|overview|status|count|breakdown|today|this week|recent)`, `(?i)(any|are there).*(new|open|unresolved).*(alert|incident)s?`, `(?i)(alert|incident|threat).*(volume|number|total)`},
			Keywords:     []string{"alerts", "incidents", "threats", "critical alerts", "open alerts", "how many alerts"},
		},
		{
			Intent:         "alert_detail",
			ToolName:       "alert_detail",
			Priority:       85,
			Description:    "Get detailed information about a specific security alert",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts:   []string{`(?i)(show|tell|get|details?|info|about).*(alert|incident)\s*#?\s*[\w-]{4,}`, `(?i)(what happened|explain|describe).*(alert|incident)\s*#?\s*[\w-]{4,}`, `(?i)alert\s*#?\s*[0-9a-f-]{8,}`, `(?i)(show|tell|get).*(details?|info|about)$`},
			Keywords:       []string{"alert details", "about alert", "what happened", "explain alert", "alert info"},
		},
		{
			Intent:         "asset_lookup",
			ToolName:       "asset_lookup",
			Priority:       70,
			Description:    "Look up details about a specific asset, server, or device",
			RequiresEntity: true,
			EntityType:     "asset_name",
			PatternTexts:   []string{`(?i)(show|tell|get|find|look\s*up).*(about|detail|info).*(asset|server|host|machine|device|endpoint)`, `(?i)(what do we know about|details?\s*(for|on|about))\s+[\w.-]+`, `(?i)(asset|server|host|device)\s+[\w.-]+\s*(info|detail|status|vuln)`, `(?i)(show|tell|get|find|look\s*up).*(web|db|host|srv|prod|dev|corp)[\w.-]*`},
			Keywords:       []string{"asset", "server", "host", "device", "machine", "details about", "look up"},
		},
		{
			Intent:       "vulnerability_query",
			ToolName:     "vulnerability_summary",
			Priority:     70,
			Description:  "View top vulnerabilities across the organization",
			PatternTexts: []string{`(?i)(top|worst|critical|unpatched|open).*(vuln|cve|weakness|exposure)s?`, `(?i)(vuln|cve|weakness).*(summary|overview|top|critical|list|count|worst)`, `(?i)(show|list|what are).*(vuln|cve)s?`},
			Keywords:     []string{"vulnerabilities", "CVEs", "unpatched", "weaknesses", "exposures", "top vulns"},
		},
		{
			Intent:       "mitre_query",
			ToolName:     "mitre_coverage",
			Priority:     75,
			Description:  "Check MITRE ATT&CK detection coverage and gaps",
			PatternTexts: []string{`(?i)mitre.*(coverage|matrix|att&?ck|detection|gap)`, `(?i)(detection|att&?ck).*(coverage|gap|matrix)`, `(?i)(what|which).*(technique|tactic).*(cover|detect|miss)`},
			Keywords:     []string{"MITRE", "ATT&CK", "coverage", "detection gaps", "techniques", "tactics"},
		},
		{
			Intent:       "ueba_query",
			ToolName:     "ueba_summary",
			Priority:     70,
			Description:  "View users and entities with anomalous behavioral patterns",
			PatternTexts: []string{`(?i)(risky|suspicious|anomal|unusual).*(user|entity|account|employee|person)s?`, `(?i)(who|which).*(user|entity|person).*(risk|suspicious|anomal|unusual)`, `(?i)(insider|behavioral).*(threat|risk|anomal)`, `(?i)ueba.*(summary|alert|risk|top)`},
			Keywords:     []string{"risky users", "suspicious activity", "behavioral", "UEBA", "insider threat", "anomalous"},
		},
		{
			Intent:       "pipeline_query",
			ToolName:     "pipeline_status",
			Priority:     65,
			Description:  "Check data pipeline health and failures",
			PatternTexts: []string{`(?i)(pipeline|etl|data flow|data job).*(status|fail|error|running|broken|down|health)`, `(?i)(any|are).*(pipeline|etl|data).*(fail|down|broken|error|stuck)`, `(?i)(data|pipeline).*(health|status|problem)`},
			Keywords:     []string{"pipeline", "ETL", "data flow", "failing", "broken", "data job", "data health"},
		},
		{
			Intent:       "compliance_query",
			ToolName:     "compliance_score",
			Priority:     70,
			Description:  "Check compliance status across regulatory frameworks",
			PatternTexts: []string{`(?i)(compliance|iso|nca|sama|nist|soc2|gdpr|hipaa|pci).*(score|status|gap|ready|progress|summary)`, `(?i)(how|are we).*(compliant|compliance|audit.ready)`, `(?i)(audit|regulatory|framework).*(status|readiness|score)`},
			Keywords:     []string{"compliance", "ISO 27001", "NCA", "SAMA", "NIST", "SOC2", "audit ready", "framework"},
		},
		{
			Intent:       "recommendation_query",
			ToolName:     "recommendation",
			Priority:     80,
			Description:  "Get personalized recommendations on what to focus on",
			PatternTexts: []string{`(?i)(what should|recommend|suggest|prioriti[sz]e|focus|action|todo)`, `(?i)(top|most important|urgent|critical).*(action|task|priority|thing|issue)s?`, `(?i)(what|where).*(should|do|focus|start).*(first|today|now|next)`, `(?i)(daily|morning|today).*(brief|summary|priority|action)`, `(?i)what.*(matters?|urgent|important|press).*(today|now|right now)`},
			Keywords:     []string{"recommend", "suggest", "prioritize", "focus", "should I", "action items", "what next", "today"},
		},
		{
			Intent:         "dashboard_build",
			ToolName:       "dashboard_builder",
			Priority:       75,
			Description:    "Build a custom dashboard with specified metrics and charts",
			RequiresEntity: true,
			EntityType:     "description",
			PatternTexts:   []string{`(?i)(build|create|make|generate|set up).*(dashboard|view|board|panel)`, `(?i)(dashboard|view|panel).*(for|about|showing|with|track)`, `(?i)(i need|give me|can you make).*(dashboard|view|chart|visual)`},
			Keywords:       []string{"build dashboard", "create view", "custom dashboard", "make dashboard", "visualization"},
		},
		{
			Intent:         "investigation_query",
			ToolName:       "investigation",
			Priority:       90,
			Description:    "Run a comprehensive investigation on a specific alert",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts:   []string{`(?i)investigat.*(alert|incident|event|threat)\s*#?\s*[\w-]{4,}`, `(?i)(deep dive|analy[sz]e|look into|dig into|examine).*(alert|incident)\s*#?\s*[\w-]{4,}`, `(?i)(full|complete|detailed).*(investigation|analysis).*(alert|incident)\s*#?\s*[\w-]{4,}`, `(?i)investigat(e|ion).*(it|that|this|first|second|third|last)`},
			Keywords:       []string{"investigate", "deep dive", "analyze", "look into", "examine", "dig into"},
		},
		{
			Intent:       "trend_query",
			ToolName:     "trend_analysis",
			Priority:     65,
			Description:  "Analyze security trends and how metrics have changed",
			PatternTexts: []string{`(?i)(how|has).*(risk|alert|score|posture|threat).*(chang|trend|evolv|compar|improv|worsen)`, `(?i)(trend|change|history|comparison|progress).*(risk|alert|score|threat|security)`, `(?i)(over time|week over week|month over month|getting better|getting worse)`, `(?i)(compar).*(this week|last week|this month|last month|yesterday)`},
			Keywords:     []string{"trend", "change", "history", "comparison", "over time", "progress", "getting better"},
		},
		{
			Intent:         "remediation_query",
			ToolName:       "remediation",
			Priority:       85,
			Description:    "Start a governed remediation action for an alert or vulnerability",
			RequiresEntity: true,
			EntityType:     "alert_id",
			PatternTexts:   []string{`(?i)(start|run|execute|trigger|initiate|begin).*(remediat|fix|patch|mitiga|resolv)`, `(?i)remediat.*(alert|vuln|issue|threat|incident)\s*#?\s*[\w-]{4,}`, `(?i)(fix|patch|block|isolate|contain).*(alert|threat|vuln)\s*#?\s*[\w-]{4,}`},
			Keywords:       []string{"remediate", "fix", "patch", "mitigate", "start remediation", "block", "isolate"},
		},
		{
			Intent:       "report_query",
			ToolName:     "report_generator",
			Priority:     70,
			Description:  "Generate an executive security report",
			PatternTexts: []string{`(?i)(generate|create|produce|write|prepare).*(report|brief|summary|executive|pdf|document)`, `(?i)(executive|security|weekly|monthly|quarterly|annual).*(report|brief|summary)`, `(?i)(need|want|give me).*(report|brief|summary)`},
			Keywords:     []string{"generate report", "executive summary", "security briefing", "weekly report", "pdf"},
		},
	})
}

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

func compileIntentPatterns(specs []intentSpec) []*chatmodel.IntentPattern {
	out := make([]*chatmodel.IntentPattern, 0, len(specs))
	for _, spec := range specs {
		patterns := make([]*regexp.Regexp, 0, len(spec.PatternTexts))
		for _, text := range spec.PatternTexts {
			patterns = append(patterns, regexp.MustCompile(text))
		}
		out = append(out, &chatmodel.IntentPattern{
			Intent:         spec.Intent,
			ToolName:       spec.ToolName,
			Patterns:       patterns,
			PatternStrings: append([]string(nil), spec.PatternTexts...),
			Keywords:       append([]string(nil), spec.Keywords...),
			RequiresEntity: spec.RequiresEntity,
			EntityType:     spec.EntityType,
			Priority:       spec.Priority,
			Description:    spec.Description,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return fmt.Sprintf("%s:%s", out[i].Intent, out[i].ToolName) < fmt.Sprintf("%s:%s", out[j].Intent, out[j].ToolName)
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}

