package model

type DiscoveredSchema struct {
	Tables             []DiscoveredTable  `json:"tables"`
	TableCount         int                `json:"table_count"`
	ColumnCount        int                `json:"column_count"`
	ContainsPII        bool               `json:"contains_pii"`
	HighestClass       DataClassification `json:"highest_classification"`
	DiscoveredWarnings []string           `json:"warnings,omitempty"`
}

type DiscoveredTable struct {
	SchemaName       string             `json:"schema_name,omitempty"`
	Name             string             `json:"name"`
	Type             string             `json:"type"`
	Comment          string             `json:"comment,omitempty"`
	Columns          []DiscoveredColumn `json:"columns"`
	PrimaryKeys      []string           `json:"primary_keys,omitempty"`
	ForeignKeys      []ForeignKey       `json:"foreign_keys,omitempty"`
	EstimatedRows    int64              `json:"estimated_rows,omitempty"`
	SizeBytes        int64              `json:"size_bytes,omitempty"`
	InferredClass    DataClassification `json:"inferred_classification"`
	ContainsPII      bool               `json:"contains_pii"`
	PIIColumnCount   int                `json:"pii_column_count"`
	NullableCount    int                `json:"nullable_count"`
	SampledRowCount  int                `json:"sampled_row_count"`
	DiscoveryWarning []string           `json:"warnings,omitempty"`
}

type DiscoveredColumn struct {
	Name              string             `json:"name"`
	DataType          string             `json:"data_type"`
	NativeType        string             `json:"native_type"`
	MappedType        string             `json:"mapped_type"`
	Subtype           string             `json:"subtype,omitempty"`
	MaxLength         *int               `json:"max_length,omitempty"`
	Nullable          bool               `json:"nullable"`
	DefaultValue      *string            `json:"default_value,omitempty"`
	Comment           string             `json:"comment,omitempty"`
	IsPrimaryKey      bool               `json:"is_primary_key"`
	IsForeignKey      bool               `json:"is_foreign_key"`
	ForeignKeyRef     *ForeignKeyRef     `json:"foreign_key_ref,omitempty"`
	SampleValues      []string           `json:"sample_values,omitempty"`
	SampleStats       SampleStats        `json:"sample_stats,omitempty"`
	InferredPII       bool               `json:"inferred_pii"`
	InferredPIIType   string             `json:"inferred_pii_type,omitempty"`
	InferredClass     DataClassification `json:"inferred_classification"`
	DetectionReasons  []string           `json:"detection_reasons,omitempty"`
}

type ForeignKey struct {
	Column        string `json:"column"`
	ReferencedRef ForeignKeyRef `json:"referenced_ref"`
}

type ForeignKeyRef struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

type SampleStats struct {
	NullCount       int      `json:"null_count"`
	DistinctCount   int      `json:"distinct_count"`
	LooksLikeEmail  bool     `json:"looks_like_email"`
	LooksLikePhone  bool     `json:"looks_like_phone"`
	LooksLikeCard   bool     `json:"looks_like_credit_card"`
	LooksLikeIP     bool     `json:"looks_like_ip"`
	EnumValues      []string `json:"enum_values,omitempty"`
	MinValue        *string  `json:"min_value,omitempty"`
	MaxValue        *string  `json:"max_value,omitempty"`
	ObservedSamples int      `json:"observed_samples"`
}
