package feed

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/cyber/cti/feed/adapters"
)

// Normalizer dispatches raw feed data to the appropriate adapter for parsing.
type Normalizer struct {
	registry map[string]adapters.FeedAdapter
}

func NewNormalizer() *Normalizer {
	n := &Normalizer{registry: make(map[string]adapters.FeedAdapter)}
	// Register built-in adapters
	for _, a := range []adapters.FeedAdapter{
		adapters.NewSTIXAdapter(),
		adapters.NewCSVAdapter(),
		adapters.NewGenericJSONAdapter(),
	} {
		n.registry[a.SourceType()] = a
	}
	return n
}

// RegisterAdapter adds a custom adapter.
func (n *Normalizer) RegisterAdapter(a adapters.FeedAdapter) {
	n.registry[a.SourceType()] = a
}

// Normalize parses raw bytes using the adapter matching sourceType.
func (n *Normalizer) Normalize(ctx context.Context, sourceType string, raw []byte) ([]adapters.NormalizedIndicator, error) {
	a, ok := n.registry[sourceType]
	if !ok {
		// Fall back to generic JSON
		a, ok = n.registry["json_generic"]
		if !ok {
			return nil, fmt.Errorf("no adapter for source type: %s", sourceType)
		}
	}
	return a.Parse(ctx, raw)
}
