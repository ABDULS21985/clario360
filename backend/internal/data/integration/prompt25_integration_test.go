//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	datadto "github.com/clario360/platform/internal/data/dto"
	datamodel "github.com/clario360/platform/internal/data/model"
	datarepo "github.com/clario360/platform/internal/data/repository"
)

type paginatedEnvelope[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

func TestPrompt25_LineageDarkDataAnalyticsDashboard_HTTP(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	sourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "prompt25-pg",
		Description: "prompt25 postgres source",
		Type:        string(datamodel.DataSourceTypePostgreSQL),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"host":     h.sourcePostgresHost,
			"port":     h.sourcePostgresPort,
			"database": "source",
			"schema":   "app",
			"username": "sourceuser",
			"password": "sourcepass",
			"ssl_mode": "disable",
		}),
	})

	schema := h.discoverSource(t, sourceID)
	if !schemaHasTable(schema, "customers") || !schemaHasTable(schema, "orders") {
		t.Fatalf("schema discovery missing expected tables")
	}

	customerModelID := h.deriveModel(t, sourceID, "customers", "customer_master")
	h.setModelState(t, customerModelID, datamodel.DataModelStatusActive, datamodel.DataClassificationRestricted)

	source := h.getSource(t, sourceID)
	customerModel := h.getModel(t, customerModelID)

	h.recordLineage(t, datadto.RecordLineageEdgeRequest{
		SourceType:   string(datamodel.LineageEntityDataSource),
		SourceID:     sourceID,
		SourceName:   source.Name,
		TargetType:   string(datamodel.LineageEntityDataModel),
		TargetID:     customerModelID,
		TargetName:   customerModel.DisplayName,
		Relationship: string(datamodel.LineageRelationshipFeeds),
		RecordedBy:   string(datamodel.LineageRecordedByManual),
	})

	suiteConsumerID := uuid.New()
	h.recordLineage(t, datadto.RecordLineageEdgeRequest{
		SourceType:   string(datamodel.LineageEntityDataModel),
		SourceID:     customerModelID,
		SourceName:   customerModel.DisplayName,
		TargetType:   string(datamodel.LineageEntitySuiteConsumer),
		TargetID:     suiteConsumerID,
		TargetName:   "cyber-dspm",
		Relationship: string(datamodel.LineageRelationshipConsumedBy),
		RecordedBy:   string(datamodel.LineageRecordedByManual),
	})

	graph := h.getLineageGraph(t, "/api/v1/data/lineage/graph")
	if graph.Stats.NodeCount < 3 || graph.Stats.EdgeCount < 2 {
		t.Fatalf("unexpected lineage graph stats: %+v", graph.Stats)
	}

	entityGraph := h.getLineageGraph(t, fmt.Sprintf("/api/v1/data/lineage/graph/data_model/%s", customerModelID))
	centerFound := false
	for _, node := range entityGraph.Nodes {
		if node.EntityID == customerModelID {
			centerFound = true
			if node.Metadata == nil || node.Metadata["is_center"] != true {
				t.Fatalf("expected model node to be marked as center: %+v", node.Metadata)
			}
		}
	}
	if !centerFound {
		t.Fatal("expected entity graph to include centered model node")
	}

	impact := h.getImpact(t, datamodel.LineageEntityDataSource, sourceID)
	if impact.TotalAffected < 2 {
		t.Fatalf("expected downstream impact, got %+v", impact)
	}
	if !containsAffectedSuite(impact.AffectedSuites, "Cybersecurity") {
		t.Fatalf("expected cybersecurity suite impact, got %+v", impact.AffectedSuites)
	}

	scan := h.runDarkDataScan(t)
	if scan.AssetsDiscovered == 0 {
		t.Fatalf("expected dark data discoveries, got %+v", scan)
	}

	darkAssets := h.listDarkDataAssets(t, "reason=unmodeled")
	orderAsset := findDarkDataAsset(darkAssets, "shadow_contacts")
	if orderAsset == nil {
		t.Fatalf("expected unmodeled shadow_contacts table, got %+v", darkAssets)
	}
	if !orderAsset.ContainsPII {
		t.Fatalf("expected shadow_contacts dark data asset to inherit pii inference from metadata")
	}

	governedModel := h.governDarkData(t, orderAsset.ID, "orders_governed")
	if governedModel.SourceID == nil || *governedModel.SourceID != sourceID {
		t.Fatalf("governed model missing expected source linkage: %+v", governedModel)
	}
	governedAsset := h.getDarkDataAsset(t, orderAsset.ID)
	if governedAsset.GovernanceStatus != datamodel.DarkDataGovernanceGoverned || governedAsset.LinkedModelID == nil {
		t.Fatalf("dark data asset was not governed: %+v", governedAsset)
	}

	analystToken := h.tokenForRoles(t, "analyst")
	restrictedResp := h.doJSONWithToken(t, analystToken, http.MethodPost, "/api/v1/data/analytics/query", datadto.ExecuteAnalyticsQueryRequest{
		ModelID: customerModelID,
		Query: datamodel.AnalyticsQuery{
			Columns: []string{"email", "ssn"},
			Limit:   10,
		},
	})
	defer restrictedResp.Body.Close()
	if restrictedResp.StatusCode != http.StatusForbidden {
		t.Fatalf("restricted analytics query status = %d, want %d, body=%s", restrictedResp.StatusCode, http.StatusForbidden, readBody(t, restrictedResp))
	}

	h.setModelState(t, customerModelID, datamodel.DataModelStatusActive, datamodel.DataClassificationInternal)

	maskedResult := h.executeAnalytics(t, analystToken, customerModelID, datamodel.AnalyticsQuery{
		Columns: []string{"email", "first_name", "ssn"},
		OrderBy: []datamodel.AnalyticsOrder{{Column: "email", Direction: "asc"}},
		Limit:   10,
	})
	if maskedResult.RowCount == 0 {
		t.Fatal("expected masked analytics query rows")
	}
	firstMasked := maskedResult.Rows[0]
	if firstMasked["email"] == "alice@example.com" || firstMasked["ssn"] == "123-45-6789" {
		t.Fatalf("expected masked pii values, got %+v", firstMasked)
	}
	if maskedResult.Metadata.PIIMaskingApplied != true || len(maskedResult.Metadata.ColumnsMasked) == 0 {
		t.Fatalf("expected pii masking metadata, got %+v", maskedResult.Metadata)
	}

	rawResult := h.executeAnalytics(t, h.token, customerModelID, datamodel.AnalyticsQuery{
		Columns: []string{"email", "ssn"},
		OrderBy: []datamodel.AnalyticsOrder{{Column: "email", Direction: "asc"}},
		Limit:   10,
	})
	if rawResult.RowCount == 0 {
		t.Fatal("expected analytics query rows for admin")
	}
	firstRaw := rawResult.Rows[0]
	if firstRaw["email"] != "alice@example.com" || firstRaw["ssn"] != "123-45-6789" {
		t.Fatalf("expected unmasked pii values for admin, got %+v", firstRaw)
	}

	savedQueryID := h.createSavedQuery(t, datadto.SaveQueryRequest{
		Name:        "customer_email_lookup",
		Description: "Saved query for prompt25 integration",
		ModelID:     customerModelID,
		Visibility:  "private",
		QueryDefinition: datamodel.AnalyticsQuery{
			Columns: []string{"email", "first_name"},
			Limit:   10,
		},
	})
	savedRun := h.runSavedQuery(t, savedQueryID)
	if savedRun.RowCount == 0 {
		t.Fatal("expected saved query run to return rows")
	}

	auditLogs := h.listAnalyticsAudit(t)
	if len(auditLogs) < 3 {
		t.Fatalf("expected analytics audit logs, got %d", len(auditLogs))
	}
	classificationSeen := false
	for _, entry := range auditLogs {
		if entry.ModelID == customerModelID && entry.DataClassification == string(datamodel.DataClassificationInternal) {
			classificationSeen = true
			break
		}
	}
	if !classificationSeen {
		t.Fatalf("expected analytics audit entry with data classification, got %+v", auditLogs)
	}

	h.seedDashboardSupportData(t, sourceID, customerModelID)

	dashboard := h.getDashboard(t)
	if dashboard.KPIs.TotalSources < 1 || dashboard.KPIs.TotalModels < 1 {
		t.Fatalf("unexpected dashboard kpis: %+v", dashboard.KPIs)
	}
	if dashboard.KPIs.OpenContradictions < 1 {
		t.Fatalf("expected open contradictions in dashboard: %+v", dashboard.KPIs)
	}
	if dashboard.DarkDataStats["total_assets"] == nil || dashboard.LineageStats["node_count"] == nil {
		t.Fatalf("expected populated dashboard sections: %+v", dashboard)
	}

	cachedDashboard := h.getDashboard(t)
	if cachedDashboard.CachedAt == nil {
		t.Fatalf("expected second dashboard call to be served from cache")
	}
}

func TestPrompt25_LineageGraph_100Nodes_Under2Seconds(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	h.seedLineagePerfGraph(t, 100, 150)

	start := time.Now()
	graph := h.getLineageGraph(t, "/api/v1/data/lineage/graph")
	duration := time.Since(start)
	if graph.Stats.NodeCount < 100 || graph.Stats.EdgeCount < 150 {
		t.Fatalf("unexpected graph size: %+v", graph.Stats)
	}
	if duration >= 2*time.Second {
		t.Fatalf("lineage graph duration = %s, want < 2s", duration)
	}
}

func TestPrompt25_DataDashboard_Under500Milliseconds(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	sourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "prompt25-dashboard-pg",
		Description: "dashboard postgres source",
		Type:        string(datamodel.DataSourceTypePostgreSQL),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"host":     h.sourcePostgresHost,
			"port":     h.sourcePostgresPort,
			"database": "source",
			"schema":   "app",
			"username": "sourceuser",
			"password": "sourcepass",
			"ssl_mode": "disable",
		}),
	})
	h.discoverSource(t, sourceID)
	modelID := h.deriveModel(t, sourceID, "customers", "dashboard_customer_master")
	h.setModelState(t, modelID, datamodel.DataModelStatusActive, datamodel.DataClassificationInternal)
	h.seedDashboardSupportData(t, sourceID, modelID)
	h.recordLineage(t, datadto.RecordLineageEdgeRequest{
		SourceType:   string(datamodel.LineageEntityDataSource),
		SourceID:     sourceID,
		SourceName:   h.getSource(t, sourceID).Name,
		TargetType:   string(datamodel.LineageEntityDataModel),
		TargetID:     modelID,
		TargetName:   h.getModel(t, modelID).DisplayName,
		Relationship: string(datamodel.LineageRelationshipFeeds),
		RecordedBy:   string(datamodel.LineageRecordedByManual),
	})
	h.runDarkDataScan(t)

	start := time.Now()
	dashboard := h.getDashboard(t)
	duration := time.Since(start)
	if dashboard.KPIs.TotalSources < 1 || dashboard.KPIs.TotalModels < 1 {
		t.Fatalf("unexpected dashboard payload: %+v", dashboard.KPIs)
	}
	if duration >= 500*time.Millisecond {
		t.Fatalf("dashboard duration = %s, want < 500ms", duration)
	}
}

func (h *integrationHarness) tokenForRoles(t *testing.T, roles ...string) string {
	t.Helper()
	tokenPair, err := h.jwtMgr.GenerateTokenPair(uuid.NewString(), h.tenantID.String(), "integration-role@clario360.test", roles)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}
	return tokenPair.AccessToken
}

func (h *integrationHarness) doJSONWithToken(t *testing.T, token, method, path string, body any) *http.Response {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(%T): %v", body, err)
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(h.ctx, method, h.httpServer.URL+path, payload)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("http.Do() error = %v", err)
	}
	return resp
}

func (h *integrationHarness) getSource(t *testing.T, sourceID uuid.UUID) datamodel.DataSource {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/sources/%s", sourceID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get source status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataSource]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) getModel(t *testing.T, modelID uuid.UUID) datamodel.DataModel {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/models/%s", modelID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get model status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataModel]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) setModelState(t *testing.T, modelID uuid.UUID, status datamodel.DataModelStatus, classification datamodel.DataClassification) {
	t.Helper()
	_, err := h.serviceDB.Exec(h.ctx, `
		UPDATE data_models
		SET status = $3, data_classification = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		h.tenantID, modelID, status, classification,
	)
	if err != nil {
		t.Fatalf("update model state: %v", err)
	}
}

func (h *integrationHarness) recordLineage(t *testing.T, req datadto.RecordLineageEdgeRequest) {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/lineage/record", req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("record lineage status = %d, want %d, body=%s", resp.StatusCode, http.StatusCreated, readBody(t, resp))
	}
}

func (h *integrationHarness) getLineageGraph(t *testing.T, path string) datamodel.LineageGraph {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, path, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get lineage graph status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.LineageGraph]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) getImpact(t *testing.T, entityType datamodel.LineageEntityType, entityID uuid.UUID) datamodel.ImpactAnalysis {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/lineage/impact/%s/%s", entityType, entityID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get impact status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.ImpactAnalysis]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) runDarkDataScan(t *testing.T) datamodel.DarkDataScan {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/dark-data/scan", map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("dark data scan status = %d, want %d, body=%s", resp.StatusCode, http.StatusAccepted, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DarkDataScan]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) listDarkDataAssets(t *testing.T, query string) []datamodel.DarkDataAsset {
	t.Helper()
	path := "/api/v1/data/dark-data"
	if strings.TrimSpace(query) != "" {
		path += "?" + query
	}
	resp := h.doJSON(t, http.MethodGet, path, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list dark data status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope paginatedEnvelope[datamodel.DarkDataAsset]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) getDarkDataAsset(t *testing.T, assetID uuid.UUID) datamodel.DarkDataAsset {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/dark-data/%s", assetID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get dark data asset status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DarkDataAsset]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) governDarkData(t *testing.T, assetID uuid.UUID, modelName string) datamodel.DataModel {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/dark-data/%s/govern", assetID), datadto.GovernDarkDataRequest{
		ModelName:          modelName,
		AssignQualityRules: true,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("govern dark data status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataModel]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) executeAnalytics(t *testing.T, token string, modelID uuid.UUID, query datamodel.AnalyticsQuery) datamodel.QueryResult {
	t.Helper()
	resp := h.doJSONWithToken(t, token, http.MethodPost, "/api/v1/data/analytics/query", datadto.ExecuteAnalyticsQueryRequest{
		ModelID: modelID,
		Query:   query,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execute analytics status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.QueryResult]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) createSavedQuery(t *testing.T, req datadto.SaveQueryRequest) uuid.UUID {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/analytics/saved", req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create saved query status = %d, want %d, body=%s", resp.StatusCode, http.StatusCreated, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.SavedQuery]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data.ID
}

func (h *integrationHarness) runSavedQuery(t *testing.T, savedQueryID uuid.UUID) datamodel.QueryResult {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/analytics/saved/%s/run", savedQueryID), map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("run saved query status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.QueryResult]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) listAnalyticsAudit(t *testing.T) []datamodel.AnalyticsAuditLog {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, "/api/v1/data/analytics/audit", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list analytics audit status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope paginatedEnvelope[datamodel.AnalyticsAuditLog]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) getDashboard(t *testing.T) datadto.DataSuiteDashboard {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, "/api/v1/data/dashboard", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get dashboard status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datadto.DataSuiteDashboard]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) seedDashboardSupportData(t *testing.T, sourceID, modelID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()

	logger := zerolog.Nop()
	pipelineRepo := datarepo.NewPipelineRepository(h.serviceDB, logger)
	runRepo := datarepo.NewPipelineRunRepository(h.serviceDB, logger)
	qualityRuleRepo := datarepo.NewQualityRuleRepository(h.serviceDB, logger)
	qualityResultRepo := datarepo.NewQualityResultRepository(h.serviceDB, logger)
	contradictionRepo := datarepo.NewContradictionRepository(h.serviceDB, logger)

	pipelineID := uuid.New()
	now := time.Now().UTC()
	targetTable := "analytics.customer_snapshot"
	pipelineItem := &datamodel.Pipeline{
		ID:          pipelineID,
		TenantID:    h.tenantID,
		Name:        "prompt25_dashboard_pipeline_" + uuid.NewString()[:8],
		Description: "integration dashboard pipeline",
		Type:        datamodel.PipelineTypeETL,
		SourceID:    sourceID,
		Config: datamodel.PipelineConfig{
			SourceTable:   "app.customers",
			TargetTable:   targetTable,
			LoadStrategy:  datamodel.LoadStrategyAppend,
			BatchSize:     100,
			Transformations: []datamodel.Transformation{},
		},
		Status:    datamodel.PipelineStatusActive,
		Tags:      []string{"integration"},
		CreatedBy: h.userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := pipelineRepo.Create(ctx, pipelineItem); err != nil {
		t.Fatalf("create pipeline: %v", err)
	}

	durationMs := int64(125)
	completedAt := now.Add(125 * time.Millisecond)
	currentPhase := string(datamodel.PipelinePhaseLoading)
	runItem := &datamodel.PipelineRun{
		ID:                 uuid.New(),
		TenantID:           h.tenantID,
		PipelineID:         pipelineID,
		Status:             datamodel.PipelineRunStatusCompleted,
		CurrentPhase:       &currentPhase,
		RecordsExtracted:   2,
		RecordsTransformed: 2,
		RecordsLoaded:      2,
		BytesRead:          512,
		BytesWritten:       256,
		QualityGateResults: []datamodel.QualityGateResult{},
		StartedAt:          now,
		CompletedAt:        &completedAt,
		DurationMs:         &durationMs,
		TriggeredBy:        datamodel.PipelineTriggerAPI,
		TriggeredByUser:    &h.userID,
		CreatedAt:          now,
	}
	if err := runRepo.Create(ctx, runItem); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}

	ruleColumn := "email"
	qualityRule := &datamodel.QualityRule{
		ID:          uuid.New(),
		TenantID:    h.tenantID,
		ModelID:     modelID,
		Name:        "email_not_null",
		Description: "Email should be present",
		RuleType:    datamodel.QualityRuleTypeNotNull,
		Severity:    datamodel.QualitySeverityHigh,
		ColumnName:  &ruleColumn,
		Config:      json.RawMessage(`{}`),
		Enabled:     true,
		CreatedBy:   h.userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := qualityRuleRepo.Create(ctx, qualityRule); err != nil {
		t.Fatalf("create quality rule: %v", err)
	}

	passRate := 100.0
	qualityResult := &datamodel.QualityResult{
		ID:             uuid.New(),
		TenantID:       h.tenantID,
		RuleID:         qualityRule.ID,
		ModelID:        modelID,
		PipelineRunID:  &runItem.ID,
		Status:         datamodel.QualityResultPassed,
		RecordsChecked: 2,
		RecordsPassed:  2,
		RecordsFailed:  0,
		PassRate:       &passRate,
		FailureSamples: json.RawMessage(`[]`),
		CheckedAt:      now,
		CreatedAt:      now,
	}
	if err := qualityResultRepo.Create(ctx, qualityResult); err != nil {
		t.Fatalf("create quality result: %v", err)
	}

	sourceName := "prompt25_dashboard_source"
	modelName := "prompt25_dashboard_model"
	contradiction := &datamodel.Contradiction{
		ID:              uuid.New(),
		TenantID:        h.tenantID,
		Type:            datamodel.ContradictionTypeSemantic,
		Severity:        datamodel.QualitySeverityHigh,
		ConfidenceScore: 0.92,
		Title:           "Customer classification mismatch",
		Description:     "Customer segment differs between sources",
		SourceA: datamodel.ContradictionSource{
			SourceID:   &sourceID,
			SourceName: sourceName,
			ModelID:    &modelID,
			ModelName:  modelName,
		},
		SourceB: datamodel.ContradictionSource{
			SourceID:   &sourceID,
			SourceName: sourceName,
			ModelID:    &modelID,
			ModelName:  modelName,
		},
		AffectedRecords:    1,
		SampleRecords:      json.RawMessage(`[{"customer_id":"11111111-1111-1111-1111-111111111111"}]`),
		ResolutionGuidance: "Review authoritative source and reconcile downstream model.",
		Status:             datamodel.ContradictionStatusDetected,
		Metadata:           json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := contradictionRepo.Create(ctx, contradiction); err != nil {
		t.Fatalf("create contradiction: %v", err)
	}
}

func (h *integrationHarness) seedLineagePerfGraph(t *testing.T, nodeCount, edgeCount int) {
	t.Helper()
	if nodeCount < 2 {
		nodeCount = 2
	}
	if edgeCount < nodeCount-1 {
		edgeCount = nodeCount - 1
	}
	nodes := make([]uuid.UUID, 0, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, uuid.New())
	}
	now := time.Now().UTC()
	inserted := 0
	for i := 0; i < nodeCount-1 && inserted < edgeCount; i++ {
		if err := h.insertLineageEdge(nodes[i], nodes[i+1], fmt.Sprintf("node-%03d", i), fmt.Sprintf("node-%03d", i+1), datamodel.LineageRelationshipDependsOn, now); err != nil {
			t.Fatalf("insert lineage edge: %v", err)
		}
		inserted++
	}
	for i := 0; inserted < edgeCount; i++ {
		sourceIndex := i % (nodeCount - 2)
		targetIndex := sourceIndex + 2 + (i % 3)
		if targetIndex >= nodeCount {
			targetIndex = nodeCount - 1
		}
		relationship := datamodel.LineageRelationshipFeeds
		if inserted%2 == 0 {
			relationship = datamodel.LineageRelationshipDerivedFrom
		}
		if err := h.insertLineageEdge(nodes[sourceIndex], nodes[targetIndex], fmt.Sprintf("node-%03d", sourceIndex), fmt.Sprintf("node-%03d", targetIndex), relationship, now); err != nil {
			t.Fatalf("insert lineage perf edge: %v", err)
		}
		inserted++
	}
}

func (h *integrationHarness) insertLineageEdge(sourceID, targetID uuid.UUID, sourceName, targetName string, relationship datamodel.LineageRelationship, seenAt time.Time) error {
	_, err := h.serviceDB.Exec(h.ctx, `
		INSERT INTO data_lineage_edges (
			id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
			relationship, recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, true, $11, $11, '{}'::jsonb, $11, $11
		)
		ON CONFLICT (tenant_id, source_type, source_id, target_type, target_id, relationship)
		DO NOTHING`,
		uuid.New(), h.tenantID, datamodel.LineageEntityExternal, sourceID, sourceName, datamodel.LineageEntityExternal, targetID, targetName,
		relationship, datamodel.LineageRecordedByManual, seenAt,
	)
	return err
}

func containsAffectedSuite(suites []datamodel.AffectedSuite, expected string) bool {
	for _, suite := range suites {
		if suite.SuiteName == expected {
			return true
		}
	}
	return false
}

func findDarkDataAsset(items []datamodel.DarkDataAsset, tableName string) *datamodel.DarkDataAsset {
	normalized := strings.TrimSpace(strings.ToLower(tableName))
	for index := range items {
		item := &items[index]
		if item.TableName != nil && strings.TrimSpace(strings.ToLower(*item.TableName)) == normalized {
			return item
		}
		if strings.Contains(strings.ToLower(item.Name), normalized) {
			return item
		}
	}
	return nil
}
