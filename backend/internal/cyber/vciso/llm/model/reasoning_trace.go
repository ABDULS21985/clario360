package model

type ReasoningStep struct {
	Step      int      `json:"step"`
	Action    string   `json:"action"`
	Detail    string   `json:"detail"`
	ToolNames []string `json:"tool_names,omitempty"`
}

type GroundingResult struct {
	Status            string             `json:"status"`
	TotalClaims       int                `json:"total_claims"`
	GroundedClaims    int                `json:"grounded_claims"`
	UngroundedClaims  []UngroundedClaim  `json:"ungrounded_claims,omitempty"`
	CorrectedResponse string             `json:"corrected_response,omitempty"`
}

type UngroundedClaim struct {
	Claim      string `json:"claim"`
	Type       string `json:"type"`
	Critical   bool   `json:"critical"`
	Suggestion string `json:"suggestion,omitempty"`
}
