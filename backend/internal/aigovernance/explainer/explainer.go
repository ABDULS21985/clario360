package explainer

import (
	"context"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type Explainer interface {
	Explain(ctx context.Context, version *aigovmodel.ModelVersion, input any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error)
}
