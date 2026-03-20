package metrics

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var validMetricName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

// Metrics provides a centralized Prometheus metric registry for a service.
// It uses a dedicated prometheus.Registry (not the global DefaultRegisterer)
// so that tests can create isolated registries without pollution.
type Metrics struct {
	registry    *prometheus.Registry
	serviceName string

	mu         sync.Mutex
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec

	HTTP  *HTTPMetrics
	DB    *DBMetrics
	Kafka *KafkaMetrics
}

// NewMetrics creates a new Metrics with a dedicated registry and all standard metric sets.
func NewMetrics(serviceName string) *Metrics {
	reg := prometheus.NewRegistry()

	// Register Go runtime and process collectors.
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Metrics{
		registry:    reg,
		serviceName: serviceName,
		counters:    make(map[string]*prometheus.CounterVec),
		gauges:      make(map[string]*prometheus.GaugeVec),
		histograms:  make(map[string]*prometheus.HistogramVec),
	}

	m.HTTP = newHTTPMetrics(reg, serviceName)
	m.DB = newDBMetrics(reg, serviceName)
	m.Kafka = newKafkaMetrics(reg, serviceName)

	return m
}

// Registry returns the underlying prometheus.Registry for advanced use cases
// (e.g., registering custom collectors).
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Handler returns an HTTP handler that serves the /metrics endpoint
// with OpenMetrics content negotiation.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Counter registers (or retrieves) a CounterVec with the given name.
// The metric name is prefixed with the service name: "{service}_{name}".
// Calling Counter with the same name twice returns the same CounterVec.
// Panics if the same name is registered with different label sets.
func (m *Metrics) Counter(name, help string, labels []string) *prometheus.CounterVec {
	fullName := m.prefixedName(name)
	validateName(fullName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.counters[fullName]; ok {
		return existing
	}

	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fullName,
		Help: help,
	}, labels)
	m.registry.MustRegister(cv)
	m.counters[fullName] = cv
	return cv
}

// Gauge registers (or retrieves) a GaugeVec with the given name.
func (m *Metrics) Gauge(name, help string, labels []string) *prometheus.GaugeVec {
	fullName := m.prefixedName(name)
	validateName(fullName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.gauges[fullName]; ok {
		return existing
	}

	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fullName,
		Help: help,
	}, labels)
	m.registry.MustRegister(gv)
	m.gauges[fullName] = gv
	return gv
}

// Histogram registers (or retrieves) a HistogramVec with the given name and buckets.
func (m *Metrics) Histogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	fullName := m.prefixedName(name)
	validateName(fullName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.histograms[fullName]; ok {
		return existing
	}

	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    fullName,
		Help:    help,
		Buckets: buckets,
	}, labels)
	m.registry.MustRegister(hv)
	m.histograms[fullName] = hv
	return hv
}

func (m *Metrics) prefixedName(name string) string {
	prefix := strings.ReplaceAll(m.serviceName, "-", "_")
	return prefix + "_" + name
}

func validateName(name string) {
	if !validMetricName.MatchString(name) {
		panic(fmt.Sprintf("metrics: invalid metric name %q", name))
	}
}
