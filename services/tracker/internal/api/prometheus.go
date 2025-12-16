package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_tracker_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "p2p_tracker_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Peer metrics
	peersOnlineGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "p2p_tracker_peers_online",
			Help: "Number of online peers",
		},
	)

	peersTotalGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "p2p_tracker_peers_total",
			Help: "Total number of registered peers",
		},
	)

	peerRegistrations = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_peer_registrations_total",
			Help: "Total number of peer registrations",
		},
	)

	// File metrics
	filesSharedGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "p2p_tracker_files_shared",
			Help: "Number of files being shared",
		},
	)

	fileAnnouncements = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_file_announcements_total",
			Help: "Total number of file announcements",
		},
	)

	fileDownloadRequests = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_file_download_requests_total",
			Help: "Total number of file download requests (peers lookup)",
		},
	)

	// Transfer metrics
	bytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_tracker_bytes_transferred_total",
			Help: "Total bytes transferred (reported by peers)",
		},
		[]string{"direction"}, // "upload" or "download"
	)

	// Relay metrics
	relayConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "p2p_tracker_relay_connections_active",
			Help: "Number of active relay connections",
		},
	)

	relayMessagesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_relay_messages_total",
			Help: "Total relay messages forwarded",
		},
	)

	// WebSocket metrics
	wsConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "p2p_tracker_ws_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	// Auth metrics
	authFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_auth_failures_total",
			Help: "Total authentication failures",
		},
	)

	rateLimitHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_tracker_rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
	)
)

// PrometheusMiddleware wraps HTTP handlers with metrics collection
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)

		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// statusResponseWriter wraps http.ResponseWriter to capture status code
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// normalizePath normalizes URL paths for metrics
func normalizePath(path string) string {
	// Normalize paths with IDs/hashes to prevent cardinality explosion
	switch {
	case len(path) > 12 && path[:12] == "/api/peers/":
		return "/api/peers/{id}"
	case len(path) > 11 && path[:11] == "/api/files/":
		if len(path) > 70 { // file hash is 64 chars
			return "/api/files/{hash}/peers"
		}
		return "/api/files/{hash}"
	default:
		return path
	}
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// UpdatePeerMetrics updates peer-related gauges
func UpdatePeerMetrics(online, total int) {
	peersOnlineGauge.Set(float64(online))
	peersTotalGauge.Set(float64(total))
}

// UpdateFileMetrics updates file-related gauges
func UpdateFileMetrics(count int) {
	filesSharedGauge.Set(float64(count))
}

// IncrementPeerRegistration increments registration counter
func IncrementPeerRegistration() {
	peerRegistrations.Inc()
}

// IncrementFileAnnouncement increments announcement counter
func IncrementFileAnnouncement() {
	fileAnnouncements.Inc()
}

// IncrementFileDownloadRequest increments download request counter
func IncrementFileDownloadRequest() {
	fileDownloadRequests.Inc()
}

// RecordBytesTransferred records bytes transferred
func RecordBytesTransferred(uploaded, downloaded int64) {
	bytesTransferred.WithLabelValues("upload").Add(float64(uploaded))
	bytesTransferred.WithLabelValues("download").Add(float64(downloaded))
}

// UpdateRelayConnections updates active relay connections
func UpdateRelayConnections(count int) {
	relayConnectionsActive.Set(float64(count))
}

// IncrementRelayMessages increments relay message counter
func IncrementRelayMessages() {
	relayMessagesTotal.Inc()
}

// UpdateWSConnections updates active WebSocket connections
func UpdateWSConnections(count int) {
	wsConnectionsActive.Set(float64(count))
}

// IncrementAuthFailures increments auth failure counter
func IncrementAuthFailures() {
	authFailures.Inc()
}

// IncrementRateLimitHits increments rate limit hit counter
func IncrementRateLimitHits() {
	rateLimitHits.Inc()
}

