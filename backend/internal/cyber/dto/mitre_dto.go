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
