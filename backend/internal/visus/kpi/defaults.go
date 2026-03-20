package kpi

import (
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
)

func DefaultDefinitions(tenantID, createdBy uuid.UUID) []model.KPIDefinition {
	return []model.KPIDefinition{
		newDefaultKPI(tenantID, createdBy, "Security Risk Score", model.KPICategorySecurity, model.KPISuiteCyber, "/risk/score", "$.data.overall_score", model.KPIUnitScore, model.KPIDirectionLowerIsBetter, 60, 80),
		newDefaultKPI(tenantID, createdBy, "Open Critical Alerts", model.KPICategorySecurity, model.KPISuiteCyber, "/alerts/count?severity=critical&status=new,acknowledged", "$.data.count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 2, 5),
		newDefaultKPI(tenantID, createdBy, "Mean Time to Respond", model.KPICategorySecurity, model.KPISuiteCyber, "/dashboard", "$.data.kpis.mttr_hours", model.KPIUnitHours, model.KPIDirectionLowerIsBetter, 4, 8),
		newDefaultKPI(tenantID, createdBy, "MITRE ATT&CK Coverage", model.KPICategorySecurity, model.KPISuiteCyber, "/mitre/coverage", "$.data.coverage_percent", model.KPIUnitPercentage, model.KPIDirectionHigherIsBetter, 50, 30),
		newDefaultKPI(tenantID, createdBy, "Data Quality Score", model.KPICategoryData, model.KPISuiteData, "/quality/score", "$.data.overall_score", model.KPIUnitPercentage, model.KPIDirectionHigherIsBetter, 85, 70),
		newDefaultKPI(tenantID, createdBy, "Pipeline Success Rate", model.KPICategoryData, model.KPISuiteData, "/dashboard", "$.data.pipeline_success_rate_30d", model.KPIUnitPercentage, model.KPIDirectionHigherIsBetter, 90, 80),
		newDefaultKPI(tenantID, createdBy, "Open Contradictions", model.KPICategoryData, model.KPISuiteData, "/contradictions/stats", "$.data.open_count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 10, 20),
		newDefaultKPI(tenantID, createdBy, "Dark Data Assets", model.KPICategoryData, model.KPISuiteData, "/dark-data/stats", "$.data.total_assets", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 15, 30),
		newDefaultKPI(tenantID, createdBy, "Governance Compliance", model.KPICategoryGovernance, model.KPISuiteActa, "/compliance/score", "$.data.score", model.KPIUnitPercentage, model.KPIDirectionHigherIsBetter, 85, 70),
		newDefaultKPI(tenantID, createdBy, "Overdue Action Items", model.KPICategoryGovernance, model.KPISuiteActa, "/action-items/stats", "$.data.overdue_count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 5, 10),
		newDefaultKPI(tenantID, createdBy, "Contracts Expiring 30d", model.KPICategoryLegal, model.KPISuiteLex, "/dashboard", "$.data.kpis.expiring_in_30_days", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 3, 5),
		newDefaultKPI(tenantID, createdBy, "High Risk Contracts", model.KPICategoryLegal, model.KPISuiteLex, "/dashboard", "$.data.kpis.high_risk_contracts", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 3, 5),
	}
}

func newDefaultKPI(tenantID, createdBy uuid.UUID, name string, category model.KPICategory, suite model.KPISuite, endpoint, valuePath string, unit model.KPIUnit, direction model.KPIDirection, warning, critical float64) model.KPIDefinition {
	return model.KPIDefinition{
		TenantID:          tenantID,
		Name:              name,
		Description:       name,
		Category:          category,
		Suite:             suite,
		QueryEndpoint:     endpoint,
		QueryParams:       map[string]any{},
		ValuePath:         valuePath,
		Unit:              unit,
		WarningThreshold:  &warning,
		CriticalThreshold: &critical,
		Direction:         direction,
		CalculationType:   model.KPICalcDirect,
		SnapshotFrequency: model.KPIFrequencyDay,
		Enabled:           true,
		IsDefault:         true,
		Tags:              []string{"default"},
		CreatedBy:         createdBy,
	}
}
