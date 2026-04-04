package frameworks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSOC2_AllTSCMapped(t *testing.T) {
	if got := len(SOC2Controls()); got < 35 {
		t.Fatalf("controls = %d, want at least 35", got)
	}
}

func TestSOC2_NoEmptyStatus(t *testing.T) {
	for _, control := range SOC2Controls() {
		if control.ImplementationStatus == "" {
			t.Fatalf("control %s has empty implementation_status", control.ID)
		}
	}
}

func TestSOC2_RefsExist(t *testing.T) {
	moduleRoot := filepath.Clean("..")
	for _, control := range SOC2Controls() {
		for _, ref := range control.ImplementationRefs {
			if ref.Path == "" {
				t.Fatalf("control %s has empty ref path", control.ID)
			}
			target := filepath.Clean(filepath.Join(moduleRoot, ref.Path))
			if _, err := os.Stat(target); err != nil {
				t.Fatalf("control %s ref %s missing: %v", control.ID, ref.Path, err)
			}
		}
	}
}
