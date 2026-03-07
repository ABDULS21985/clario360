package scanner

import (
	"context"

	"github.com/clario360/platform/internal/cyber/model"
)

// Scanner is the interface all discovery scanners must implement.
type Scanner interface {
	// Type returns the scan type constant for this scanner.
	Type() model.ScanType
	// Scan executes a discovery scan with the given configuration.
	Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error)
}

// Registry holds registered scanners indexed by ScanType.
type Registry struct {
	scanners map[model.ScanType]Scanner
}

// NewRegistry creates an empty scanner registry.
func NewRegistry() *Registry {
	return &Registry{scanners: make(map[model.ScanType]Scanner)}
}

// Register adds a scanner to the registry.
func (r *Registry) Register(s Scanner) {
	r.scanners[s.Type()] = s
}

// Get returns the scanner for the given type, or nil if not found.
func (r *Registry) Get(t model.ScanType) Scanner {
	return r.scanners[t]
}
