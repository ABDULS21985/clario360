package explainer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type TemplateExplainer struct{}

func NewTemplateExplainer() *TemplateExplainer {
	return &TemplateExplainer{}
}

func (e *TemplateExplainer) Explain(_ context.Context, version *aigovmodel.ModelVersion, _ any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error) {
	meta := copyMap(output.Metadata)
	factors := make([]aigovmodel.Factor, 0, 4)
	for key, value := range meta {
		if len(factors) >= 4 {
			break
		}
		switch typed := value.(type) {
		case string:
			factors = append(factors, aigovmodel.Factor{
				Name:        key,
				Value:       typed,
				Impact:      0.1,
				Direction:   "positive",
				Description: "Input characteristic used by the template-based model.",
			})
		case float64:
			factors = append(factors, aigovmodel.Factor{
				Name:        key,
				Value:       fmt.Sprintf("%.2f", typed),
				Impact:      0.1,
				Direction:   "positive",
				Description: "Numeric input factor considered by the template-driven output.",
			})
		}
	}

	human, err := renderTemplate(version, map[string]any{
		"metadata":   meta,
		"confidence": output.Confidence,
	})
	if err != nil {
		return nil, err
	}
	if human == "" {
		human = "The output was produced by deterministic template rendering over structured meeting or document context."
	}

	return &aigovmodel.Explanation{
		Structured:    meta,
		HumanReadable: human,
		Factors:       factors,
		Confidence:    output.Confidence,
		ExplainerType: string(aigovmodel.ExplainabilityTemplateBased),
		ModelSlug:     version.ModelSlug,
		ModelVersion:  version.VersionNumber,
	}, nil
}
