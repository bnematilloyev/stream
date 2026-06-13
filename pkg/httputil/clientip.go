package httputil

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the client IP without port, safe for PostgreSQL INET columns.
func ClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); xff != "" {
		return parseHost(xff)
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return parseHost(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return parseHost(r.RemoteAddr)
	}
	return host
}

func parseHost(value string) string {
	value = strings.TrimSpace(value)
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return value
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}
