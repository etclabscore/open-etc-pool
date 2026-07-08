package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The /metrics endpoint replaces the old New Relic agent. Verify it serves
// Prometheus output including Go runtime telemetry (the process-level metrics
// the agent used to collect).
func TestMetricsHandler(t *testing.T) {
	srv := httptest.NewServer(metricsHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "go_goroutines") {
		t.Error("/metrics must expose Go runtime metrics (go_goroutines)")
	}
}
