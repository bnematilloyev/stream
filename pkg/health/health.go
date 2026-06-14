package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const defaultTimeout = 2 * time.Second

// Checker validates a single dependency (Strategy pattern).
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// Report is the JSON payload for readiness endpoints.
type Report struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Evaluate runs all checkers and returns HTTP status + report.
func Evaluate(ctx context.Context, checkers ...Checker) (int, Report) {
	checks := make(map[string]string, len(checkers))
	status := http.StatusOK

	for _, c := range checkers {
		checkCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
		err := c.Check(checkCtx)
		cancel()

		if err != nil {
			checks[c.Name()] = err.Error()
			status = http.StatusServiceUnavailable
		} else {
			checks[c.Name()] = "ok"
		}
	}

	reportStatus := "ready"
	if status != http.StatusOK {
		reportStatus = "not_ready"
	}

	return status, Report{Status: reportStatus, Checks: checks}
}

// WriteJSON encodes a health report as JSON.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// Liveness responds with a simple ok status (no dependency checks).
func Liveness(w http.ResponseWriter, service string) {
	WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": service,
	})
}

// Readiness runs checkers and writes the readiness response.
func Readiness(w http.ResponseWriter, r *http.Request, checkers ...Checker) {
	status, report := Evaluate(r.Context(), checkers...)
	WriteJSON(w, status, report)
}
