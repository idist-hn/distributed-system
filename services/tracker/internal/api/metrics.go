package api

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds Prometheus-style metrics for the tracker
type Metrics struct {
	mu sync.RWMutex

	// Counters
	requestsTotal    map[string]*atomic.Int64 // by endpoint
	requestsSuccess  *atomic.Int64
	requestsError    *atomic.Int64
	peersRegistered  *atomic.Int64
	peersLeft        *atomic.Int64
	filesAnnounced   *atomic.Int64

	// Gauges (updated periodically)
	peersOnline int64
	filesCount  int64

	// Server info
	startTime time.Time
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		requestsTotal:   make(map[string]*atomic.Int64),
		requestsSuccess: &atomic.Int64{},
		requestsError:   &atomic.Int64{},
		peersRegistered: &atomic.Int64{},
		peersLeft:       &atomic.Int64{},
		filesAnnounced:  &atomic.Int64{},
		startTime:       time.Now(),
	}
}

// RecordRequest records a request to an endpoint
func (m *Metrics) RecordRequest(endpoint string, success bool) {
	m.mu.Lock()
	if _, ok := m.requestsTotal[endpoint]; !ok {
		m.requestsTotal[endpoint] = &atomic.Int64{}
	}
	counter := m.requestsTotal[endpoint]
	m.mu.Unlock()

	counter.Add(1)
	if success {
		m.requestsSuccess.Add(1)
	} else {
		m.requestsError.Add(1)
	}
}

// RecordPeerRegistered records a peer registration
func (m *Metrics) RecordPeerRegistered() {
	m.peersRegistered.Add(1)
}

// RecordPeerLeft records a peer leaving
func (m *Metrics) RecordPeerLeft() {
	m.peersLeft.Add(1)
}

// RecordFileAnnounced records a file announcement
func (m *Metrics) RecordFileAnnounced() {
	m.filesAnnounced.Add(1)
}

// UpdateGauges updates gauge values
func (m *Metrics) UpdateGauges(peersOnline, filesCount int64) {
	m.mu.Lock()
	m.peersOnline = peersOnline
	m.filesCount = filesCount
	m.mu.Unlock()
}

// Handler returns an HTTP handler for /metrics endpoint
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		// Uptime
		uptime := time.Since(m.startTime).Seconds()
		fmt.Fprintf(w, "# HELP tracker_uptime_seconds Uptime of the tracker server\n")
		fmt.Fprintf(w, "# TYPE tracker_uptime_seconds gauge\n")
		fmt.Fprintf(w, "tracker_uptime_seconds %.2f\n\n", uptime)

		// Gauges
		fmt.Fprintf(w, "# HELP tracker_peers_online Current number of online peers\n")
		fmt.Fprintf(w, "# TYPE tracker_peers_online gauge\n")
		fmt.Fprintf(w, "tracker_peers_online %d\n\n", m.peersOnline)

		fmt.Fprintf(w, "# HELP tracker_files_count Current number of files\n")
		fmt.Fprintf(w, "# TYPE tracker_files_count gauge\n")
		fmt.Fprintf(w, "tracker_files_count %d\n\n", m.filesCount)

		// Counters
		fmt.Fprintf(w, "# HELP tracker_requests_total Total number of requests\n")
		fmt.Fprintf(w, "# TYPE tracker_requests_total counter\n")
		for endpoint, counter := range m.requestsTotal {
			fmt.Fprintf(w, "tracker_requests_total{endpoint=\"%s\"} %d\n", endpoint, counter.Load())
		}
		fmt.Fprintln(w)

		fmt.Fprintf(w, "# HELP tracker_requests_success Total successful requests\n")
		fmt.Fprintf(w, "# TYPE tracker_requests_success counter\n")
		fmt.Fprintf(w, "tracker_requests_success %d\n\n", m.requestsSuccess.Load())

		fmt.Fprintf(w, "# HELP tracker_requests_error Total error requests\n")
		fmt.Fprintf(w, "# TYPE tracker_requests_error counter\n")
		fmt.Fprintf(w, "tracker_requests_error %d\n\n", m.requestsError.Load())

		fmt.Fprintf(w, "# HELP tracker_peers_registered_total Total peers registered\n")
		fmt.Fprintf(w, "# TYPE tracker_peers_registered_total counter\n")
		fmt.Fprintf(w, "tracker_peers_registered_total %d\n\n", m.peersRegistered.Load())

		fmt.Fprintf(w, "# HELP tracker_files_announced_total Total files announced\n")
		fmt.Fprintf(w, "# TYPE tracker_files_announced_total counter\n")
		fmt.Fprintf(w, "tracker_files_announced_total %d\n", m.filesAnnounced.Load())
	}
}

