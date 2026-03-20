package rules

type CheckResult struct {
	Status         string                   `json:"status"`
	RecordsChecked int64                    `json:"records_checked"`
	RecordsPassed  int64                    `json:"records_passed"`
	RecordsFailed  int64                    `json:"records_failed"`
	PassRate       float64                  `json:"pass_rate"`
	FailureSamples []map[string]interface{} `json:"failure_samples"`
	FailureSummary string                   `json:"failure_summary"`
	MetricValue    float64                  `json:"metric_value"`
	Threshold      float64                  `json:"threshold"`
}

