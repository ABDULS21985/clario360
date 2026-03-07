package rules

import (
	"context"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type ConfigDecryptor interface {
	Decrypt(ciphertext []byte, keyID string) ([]byte, error)
}

type Dataset struct {
	Rule           *model.QualityRule
	Model          *model.DataModel
	Source         *repository.SourceRecord
	Rows           []map[string]interface{}
	PreviousResult *model.QualityResult
}

type Checker interface {
	Type() string
	Check(ctx context.Context, dataset Dataset) (*CheckResult, error)
}

type LiveConnectorChecker interface {
	Checker
	WithDependencies(registry *connector.ConnectorRegistry, sourceRepo *repository.SourceRepository, decryptor ConfigDecryptor) Checker
}

