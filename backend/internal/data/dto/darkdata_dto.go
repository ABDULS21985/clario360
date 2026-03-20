package dto

type ListDarkDataParams struct {
	Page             int
	PerPage          int
	Search           string
	Reason           string
	AssetType        string
	GovernanceStatus string
	ContainsPII      *bool
	MinRiskScore     *float64
	Sort             string
	Order            string
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
