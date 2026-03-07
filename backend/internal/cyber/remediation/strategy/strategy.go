package strategy

import (
	"context"
	"encoding/json"

	"github.com/clario360/platform/internal/cyber/model"
)

// RemediationStrategy is implemented by each remediation type.
type RemediationStrategy interface {
	Type() model.RemediationType
	DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error)
	Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error)
	Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error)
	Rollback(ctx context.Context, action *model.RemediationAction) error
	CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error)
}
