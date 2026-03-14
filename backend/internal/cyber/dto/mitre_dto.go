package dto

// MITRETacticDTO is returned by the tactics endpoint.
type MITRETacticDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
}

// MITRETechniqueDTO is returned by the techniques endpoint.
type MITRETechniqueDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TacticIDs   []string `json:"tactic_ids"`
	Platforms   []string `json:"platforms"`
	DataSources []string `json:"data_sources"`
}

// MITRECoverageDTO returns rule coverage for a single technique.
type MITRECoverageDTO struct {
	TechniqueID   string   `json:"technique_id"`
	TechniqueName string   `json:"technique_name"`
	TacticIDs     []string `json:"tactic_ids"`
	HasDetection  bool     `json:"has_detection"`
	RuleCount     int      `json:"rule_count"`
	RuleNames     []string `json:"rule_names"`
}

// MITRETacticCoverageDTO is a tactic with its coverage count.
type MITRETacticCoverageDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ShortName      string `json:"short_name,omitempty"`
	TechniqueCount int    `json:"technique_count"`
	CoveredCount   int    `json:"covered_count"`
}

// MITRECoverageResponseDTO is the full aggregated coverage response.
type MITRECoverageResponseDTO struct {
	Tactics            []MITRETacticCoverageDTO `json:"tactics"`
	Techniques         []MITRECoverageDTO       `json:"techniques"`
	TotalTechniques    int                      `json:"total_techniques"`
	CoveredTechniques  int                      `json:"covered_techniques"`
	CoveragePercent    float64                  `json:"coverage_percent"`
	ActiveTechniques   int                      `json:"active_techniques"`
	PassiveTechniques  int                      `json:"passive_techniques"`
}
