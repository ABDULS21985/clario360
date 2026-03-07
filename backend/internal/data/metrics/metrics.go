package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	Registry *prometheus.Registry
}

func New() *Metrics {
	return &Metrics{Registry: prometheus.NewRegistry()}
}
