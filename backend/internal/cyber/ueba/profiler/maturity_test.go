package profiler

import (
	"testing"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestMaturityProfiles(t *testing.T) {
	cases := []struct {
		name string
		obs  int64
		days int
		want model.ProfileMaturity
	}{
		{name: "learning", obs: 50, days: 10, want: model.ProfileMaturityLearning},
		{name: "baseline", obs: 500, days: 45, want: model.ProfileMaturityBaseline},
		{name: "mature", obs: 2000, days: 120, want: model.ProfileMaturityMature},
		{name: "low count long time", obs: 50, days: 100, want: model.ProfileMaturityLearning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyMaturity(tc.obs, tc.days); got != tc.want {
				t.Fatalf("ClassifyMaturity(%d, %d) = %s, want %s", tc.obs, tc.days, got, tc.want)
			}
		})
	}
}
