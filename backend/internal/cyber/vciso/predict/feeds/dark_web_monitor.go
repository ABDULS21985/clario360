package feeds

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DarkWebMention struct {
	Source     string    `json:"source"`
	Category   string    `json:"category"`
	Keyword    string    `json:"keyword"`
	Confidence float64   `json:"confidence"`
	ObservedAt time.Time `json:"observed_at"`
}

type DarkWebProvider interface {
	ListMentions(ctx context.Context, tenantID uuid.UUID) ([]DarkWebMention, error)
}

type DarkWebMonitor struct {
	provider DarkWebProvider
}

func NewDarkWebMonitor(provider DarkWebProvider) *DarkWebMonitor {
	return &DarkWebMonitor{provider: provider}
}

func (m *DarkWebMonitor) Mentions(ctx context.Context, tenantID uuid.UUID) ([]DarkWebMention, error) {
	if m == nil || m.provider == nil {
		return nil, nil
	}
	return m.provider.ListMentions(ctx, tenantID)
}

func (m *DarkWebMonitor) RiskScore(ctx context.Context, tenantID uuid.UUID, since time.Time) (float64, error) {
	items, err := m.Mentions(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	score := 0.0
	for _, item := range items {
		if item.ObservedAt.Before(since) {
			continue
		}
		score += item.Confidence
	}
	return score, nil
}
