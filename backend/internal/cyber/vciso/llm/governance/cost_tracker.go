package governance

type CostTracker struct{}

func NewCostTracker() *CostTracker {
	return &CostTracker{}
}

func (c *CostTracker) Estimate(cost float64) float64 {
	if cost < 0 {
		return 0
	}
	return cost
}
