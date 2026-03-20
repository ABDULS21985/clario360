package dto

// LineageGraphParams controls lineage graph queries.
type LineageGraphParams struct {
	Classification *string `json:"classification,omitempty"`
	EdgeType       *string `json:"edge_type,omitempty"`
	ShowInferred   *bool   `json:"show_inferred,omitempty"`
	PIIOnly        *bool   `json:"pii_only,omitempty"`
}

// TraversalParams controls upstream/downstream traversal.
type TraversalParams struct {
	Depth int `json:"depth"`
}

// SetDefaults applies default values to traversal params.
func (p *TraversalParams) SetDefaults() {
	if p.Depth < 1 || p.Depth > 10 {
		p.Depth = 3
	}
}
