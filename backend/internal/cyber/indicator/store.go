package indicator

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Refresher wraps the matcher with a periodic refresh loop.
type Refresher struct {
	matcher *Matcher
}

// NewRefresher creates a periodic indicator refresh helper.
func NewRefresher(matcher *Matcher) *Refresher {
	return &Refresher{matcher: matcher}
}

// StartRefreshLoop keeps a tenant's IOC cache up to date until the context is cancelled.
func (r *Refresher) StartRefreshLoop(ctx context.Context, tenantID uuid.UUID, interval time.Duration) {
	if r == nil || r.matcher == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.matcher.Load(context.Background(), tenantID)
			}
		}
	}()
}
