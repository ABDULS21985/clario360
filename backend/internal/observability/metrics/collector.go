package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// PgxPoolCollector implements prometheus.Collector for pgxpool.Stat().
// It reports live connection pool statistics on each Prometheus scrape.
type PgxPoolCollector struct {
	pool        *pgxpool.Pool
	serviceName string

	activeDesc       *prometheus.Desc
	idleDesc         *prometheus.Desc
	maxDesc          *prometheus.Desc
	totalCreatedDesc *prometheus.Desc
	waitCountDesc    *prometheus.Desc
	waitDurationDesc *prometheus.Desc
}

// NewPgxPoolCollector creates a collector that scrapes pgxpool statistics.
func NewPgxPoolCollector(pool *pgxpool.Pool, serviceName string) *PgxPoolCollector {
	labels := []string{"service"}
	return &PgxPoolCollector{
		pool:        pool,
		serviceName: serviceName,
		activeDesc: prometheus.NewDesc(
			"db_pool_connections_active",
			"Number of currently acquired connections.",
			labels, nil,
		),
		idleDesc: prometheus.NewDesc(
			"db_pool_connections_idle",
			"Number of idle connections in the pool.",
			labels, nil,
		),
		maxDesc: prometheus.NewDesc(
			"db_pool_connections_max",
			"Maximum number of connections allowed.",
			labels, nil,
		),
		totalCreatedDesc: prometheus.NewDesc(
			"db_pool_connections_total_created",
			"Total number of connections created over the lifetime of the pool.",
			labels, nil,
		),
		waitCountDesc: prometheus.NewDesc(
			"db_pool_connections_wait_total",
			"Total number of successful acquires that had to wait for a connection.",
			labels, nil,
		),
		waitDurationDesc: prometheus.NewDesc(
			"db_pool_connections_wait_duration_seconds_total",
			"Total time spent waiting to acquire a connection.",
			labels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *PgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.activeDesc
	ch <- c.idleDesc
	ch <- c.maxDesc
	ch <- c.totalCreatedDesc
	ch <- c.waitCountDesc
	ch <- c.waitDurationDesc
}

// Collect implements prometheus.Collector. Calls pool.Stat() and emits current values.
func (c *PgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	stat := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(c.activeDesc, prometheus.GaugeValue,
		float64(stat.AcquiredConns()), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue,
		float64(stat.IdleConns()), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.maxDesc, prometheus.GaugeValue,
		float64(stat.MaxConns()), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.totalCreatedDesc, prometheus.CounterValue,
		float64(stat.NewConnsCount()), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.waitCountDesc, prometheus.CounterValue,
		float64(stat.EmptyAcquireCount()), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.waitDurationDesc, prometheus.CounterValue,
		stat.AcquireDuration().Seconds(), c.serviceName)
}

// RedisPoolCollector implements prometheus.Collector for redis.PoolStats().
type RedisPoolCollector struct {
	client      *redis.Client
	serviceName string

	activeDesc   *prometheus.Desc
	idleDesc     *prometheus.Desc
	staleDesc    *prometheus.Desc
	hitsDesc     *prometheus.Desc
	missesDesc   *prometheus.Desc
	timeoutsDesc *prometheus.Desc
}

// NewRedisPoolCollector creates a collector that scrapes Redis pool statistics.
func NewRedisPoolCollector(client *redis.Client, serviceName string) *RedisPoolCollector {
	labels := []string{"service"}
	return &RedisPoolCollector{
		client:      client,
		serviceName: serviceName,
		activeDesc: prometheus.NewDesc(
			"redis_pool_connections_active",
			"Number of active Redis connections.",
			labels, nil,
		),
		idleDesc: prometheus.NewDesc(
			"redis_pool_connections_idle",
			"Number of idle Redis connections.",
			labels, nil,
		),
		staleDesc: prometheus.NewDesc(
			"redis_pool_connections_stale",
			"Number of stale Redis connections.",
			labels, nil,
		),
		hitsDesc: prometheus.NewDesc(
			"redis_pool_hits_total",
			"Total number of pool hits (reused connections).",
			labels, nil,
		),
		missesDesc: prometheus.NewDesc(
			"redis_pool_misses_total",
			"Total number of pool misses (new connections created).",
			labels, nil,
		),
		timeoutsDesc: prometheus.NewDesc(
			"redis_pool_timeouts_total",
			"Total number of pool timeouts.",
			labels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *RedisPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.activeDesc
	ch <- c.idleDesc
	ch <- c.staleDesc
	ch <- c.hitsDesc
	ch <- c.missesDesc
	ch <- c.timeoutsDesc
}

// Collect implements prometheus.Collector. Calls client.PoolStats() and emits values.
func (c *RedisPoolCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.client.PoolStats()

	ch <- prometheus.MustNewConstMetric(c.activeDesc, prometheus.GaugeValue,
		float64(stats.TotalConns-stats.IdleConns), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue,
		float64(stats.IdleConns), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.staleDesc, prometheus.GaugeValue,
		float64(stats.StaleConns), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.hitsDesc, prometheus.CounterValue,
		float64(stats.Hits), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.missesDesc, prometheus.CounterValue,
		float64(stats.Misses), c.serviceName)
	ch <- prometheus.MustNewConstMetric(c.timeoutsDesc, prometheus.CounterValue,
		float64(stats.Timeouts), c.serviceName)
}
