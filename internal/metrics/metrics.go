package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"videostreamgo/internal/cache"
)

// Metrics holds all Prometheus metrics for the service
type Metrics struct {
	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge

	// Cache metrics
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter
	cacheErrors prometheus.Counter

	// Database metrics
	dbConnectionsActive    prometheus.Gauge
	dbConnectionsIdle      prometheus.Gauge
	dbConnectionsWaitCount prometheus.Counter

	// Redis metrics
	redisConnectionsActive prometheus.Gauge
	redisPingDuration      prometheus.Histogram

	// Business metrics
	uploadsTotal         prometheus.Counter
	authEventsTotal      *prometheus.CounterVec
	videoProcessingTotal prometheus.Counter

	// Service info
	serviceInfo *prometheus.GaugeVec

	mu sync.RWMutex
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(serviceName, serviceVersion string) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_requests_total", serviceName),
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    fmt.Sprintf("%s_request_duration_seconds", serviceName),
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_requests_in_flight", serviceName),
				Help: "Number of HTTP requests currently being processed",
			},
		),
		cacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_cache_hits_total", serviceName),
				Help: "Total number of cache hits",
			},
		),
		cacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_cache_misses_total", serviceName),
				Help: "Total number of cache misses",
			},
		),
		cacheErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_cache_errors_total", serviceName),
				Help: "Total number of cache errors",
			},
		),
		dbConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_db_connections_active", serviceName),
				Help: "Number of active database connections",
			},
		),
		dbConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_db_connections_idle", serviceName),
				Help: "Number of idle database connections",
			},
		),
		dbConnectionsWaitCount: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_db_connections_wait_total", serviceName),
				Help: "Total number of connection wait events",
			},
		),
		redisConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_redis_connections_active", serviceName),
				Help: "Number of active Redis connections",
			},
		),
		redisPingDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    fmt.Sprintf("%s_redis_ping_duration_seconds", serviceName),
				Help:    "Redis ping duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		uploadsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_uploads_total", serviceName),
				Help: "Total number of video uploads",
			},
		),
		authEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_auth_events_total", serviceName),
				Help: "Total number of authentication events",
			},
			[]string{"event", "status"},
		),
		videoProcessingTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_video_processing_total", serviceName),
				Help: "Total number of video processing jobs",
			},
		),
		serviceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_info", serviceName),
				Help: "Service information",
			},
			[]string{"version", "service"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.cacheHits,
		m.cacheMisses,
		m.cacheErrors,
		m.dbConnectionsActive,
		m.dbConnectionsIdle,
		m.dbConnectionsWaitCount,
		m.redisConnectionsActive,
		m.redisPingDuration,
		m.uploadsTotal,
		m.authEventsTotal,
		m.videoProcessingTotal,
		m.serviceInfo,
	)

	// Set service info
	m.serviceInfo.WithLabelValues(serviceVersion, serviceName).Set(1)

	return m
}

// MetricsMiddleware returns a middleware that records request metrics
func (m *Metrics) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		// Record request duration
		m.requestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)

		// Record request count
		m.requestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			fmt.Sprintf("%d", c.Writer.Status()),
		).Inc()
	}
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit() {
	m.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss() {
	m.cacheMisses.Inc()
}

// RecordCacheError records a cache error
func (m *Metrics) RecordCacheError() {
	m.cacheErrors.Inc()
}

// UpdateCacheStats updates cache metrics from CacheStats
func (m *Metrics) UpdateCacheStats(stats cache.CacheStats) {
	m.cacheHits.Add(float64(stats.Hits))
	m.cacheMisses.Add(float64(stats.Misses))
}

// UpdateDBStats updates database connection metrics
func (m *Metrics) UpdateDBStats(openConns, inUse, idle int, waitCount int64) {
	m.dbConnectionsActive.Set(float64(openConns))
	m.dbConnectionsIdle.Set(float64(idle))
	m.dbConnectionsWaitCount.Add(float64(waitCount))
}

// RecordRedisPing records Redis ping duration
func (m *Metrics) RecordRedisPing(duration time.Duration) {
	m.redisPingDuration.Observe(duration.Seconds())
}

// RecordUpload records a video upload
func (m *Metrics) RecordUpload() {
	m.uploadsTotal.Inc()
}

// RecordAuthEvent records an authentication event
func (m *Metrics) RecordAuthEvent(event, status string) {
	m.authEventsTotal.WithLabelValues(event, status).Inc()
}

// RecordVideoProcessing records a video processing job
func (m *Metrics) RecordVideoProcessing() {
	m.videoProcessingTotal.Inc()
}

// Handler returns the Prometheus metrics handler
func Handler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// StatusRecorder is a middleware that records the status code for metrics
type StatusRecorder struct {
	gin.ResponseWriter
	status int
}

// WriteHeader captures the status code
func (w *StatusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// StatusRecorderMiddleware returns a middleware that records the status code
func StatusRecorderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		recorder := &StatusRecorder{
			ResponseWriter: c.Writer,
			status:         200,
		}
		c.Writer = recorder
		c.Next()
		// Status is now available in recorder.status
	}
}

// CollectMetricsEndpoint returns a handler that collects and reports all metrics
func (m *Metrics) CollectMetricsEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Force garbage collection for accurate memory metrics
		// Only do this periodically, not on every request
		if time.Now().Second() == 0 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			// Memory metrics are automatically collected by Prometheus
		}
		Handler()(c)
	}
}
