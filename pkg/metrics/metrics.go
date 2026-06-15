package metrics

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed",
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path"},
	)
)

// Middleware records Prometheus metrics for HTTP handlers.
func Middleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)

			path := normalizePath(r.URL.Path)
			status := strconv.Itoa(ww.status)

			httpRequestsTotal.WithLabelValues(serviceName, r.Method, path, status).Inc()
			httpRequestDuration.WithLabelValues(serviceName, r.Method, path).Observe(time.Since(start).Seconds())
		})
	}
}

// Handler exposes the Prometheus scrape endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// normalizePath reduces metric cardinality by replacing dynamic segments.
func normalizePath(path string) string {
	switch {
	case path == "/health", path == "/ready", path == "/metrics":
		return path
	case len(path) >= 4 && path[:4] == "/v1/":
		return normalizeAPIPath(path)
	default:
		return path
	}
}

func normalizeAPIPath(path string) string {
	parts := splitPath(path)
	if len(parts) < 3 {
		return path
	}
	// /v1/{resource}/...
	switch parts[1] {
	case "users", "channels", "streams":
		if len(parts) >= 4 && parts[3] != "" {
			parts[3] = ":id"
		}
		if len(parts) >= 5 && parts[4] != "" {
			parts[4] = ":action"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	parts := make([]string, 0, len(raw)+1)
	parts = append(parts, "")
	for _, p := range raw {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func joinPath(parts []string) string {
	if len(parts) <= 1 {
		return "/"
	}
	return "/" + strings.Join(parts[1:], "/")
}
