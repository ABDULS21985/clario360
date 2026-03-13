package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type AssetRiskPredictionTool struct {
	baseTool
}

func NewAssetRiskPredictionTool(deps *Dependencies) *AssetRiskPredictionTool {
	return &AssetRiskPredictionTool{baseTool: newBaseTool(deps)}
}

func (t *AssetRiskPredictionTool) Name() string { return "get_asset_risk_prediction" }

func (t *AssetRiskPredictionTool) Description() string {
	return "get predicted probability of each asset being targeted, ranked by risk"
}

func (t *AssetRiskPredictionTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *AssetRiskPredictionTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.PredictEngine == nil {
		return nil, fmt.Errorf("%w: predictive engine", errToolUnavailable)
	}
	limit := t.parseCount(map[string]string{"count": firstNonEmpty(params["limit"], params["count"])}, 10, 25)
	assetType := strings.TrimSpace(strings.ToLower(params["asset_type"]))
	response, err := t.deps.PredictEngine.PredictAssetRisk(ctx, tenantID, limit, assetType)
	if err != nil {
		return nil, err
	}
	lines := []string{fmt.Sprintf("These are the most likely assets to be targeted in the next 30 days (top %d):", len(response.Items)), ""}
	entities := make([]chatmodel.EntityReference, 0, len(response.Items))
	rows := make([]map[string]any, 0, len(response.Items))
	for idx, item := range response.Items {
		lines = append(lines, fmt.Sprintf("%d. %s — %.0f%% target probability (P10 %.0f%% / P90 %.0f%%)", idx+1, item.AssetName, item.Probability*100, item.Confidence.P10*100, item.Confidence.P90*100))
		entities = append(entities, entityRef("asset", item.AssetID.String(), item.AssetName, idx))
		rows = append(rows, map[string]any{
			"asset_id":     item.AssetID,
			"asset_name":   item.AssetName,
			"asset_type":   item.AssetType,
			"probability":  item.Probability,
			"current_risk": item.CurrentRisk,
		})
	}
	return makeListResult(strings.Join(lines, "\n"), map[string]any{"items": rows, "meta": response.GenericPredictionResponse}, []chatmodel.SuggestedAction{
		messageAction("Prioritize vulnerability patching", "Which of our open CVEs should we prioritize first?"),
	}, entities), nil
}
