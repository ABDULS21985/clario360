package service

import (
	"testing"

	actametrics "github.com/clario360/platform/internal/acta/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordMinutesTransition(t *testing.T) {
	metrics := actametrics.New(prometheus.NewRegistry())
	svc := &MinutesService{metrics: metrics}

	svc.recordMinutesTransition("", "draft")
	svc.recordMinutesTransition("draft", "review")

	if got := testutil.ToFloat64(metrics.MinutesStatus.WithLabelValues("draft")); got != 0 {
		t.Fatalf("draft gauge = %v, want 0", got)
	}
	if got := testutil.ToFloat64(metrics.MinutesStatus.WithLabelValues("review")); got != 1 {
		t.Fatalf("review gauge = %v, want 1", got)
	}
}
