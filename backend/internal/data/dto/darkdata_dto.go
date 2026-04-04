package dto

type ListDarkDataParams struct {
	Page               int
	PerPage            int
	Search             string
	Reasons            []string
	AssetTypes         []string
	GovernanceStatuses []string
	ContainsPII        *bool
	MinRiskScore       *float64
	Sort               string
	Order              string
}

type UpdateDarkDataStatusRequest struct {
	GovernanceStatus string  `json:"governance_status"`
	GovernanceNotes  *string `json:"governance_notes,omitempty"`
}

type GovernDarkDataRequest struct {
	ModelName          string `json:"model_name"`
	AssignQualityRules bool   `json:"assign_quality_rules"`
}

type ListDarkDataScansParams struct {
	Page    int
	PerPage int
	Status  string
}
