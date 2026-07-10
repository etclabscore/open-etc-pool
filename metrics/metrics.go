// Package metrics declares the pool's Prometheus collectors. They register with
// the default registry, so the existing /metrics endpoint (main.go) exposes them
// alongside the Go runtime and process telemetry.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Share outcome labels for ShareOutcomes.
const (
	ShareValid     = "valid"
	ShareInvalid   = "invalid"
	ShareStale     = "stale"
	ShareDuplicate = "duplicate"
	ShareMalformed = "malformed"
)

// Block submission result labels for BlockSubmissions.
const (
	SubmitAccepted = "accepted"
	SubmitRejected = "rejected"
	SubmitError    = "error"
)

var (
	// ShareOutcomes counts submitted shares by outcome. The buckets are mutually
	// exclusive: a share is malformed, stale, a duplicate, invalid, or valid.
	ShareOutcomes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oep_shares_total",
		Help: "Submitted shares by outcome (valid, invalid, stale, duplicate, malformed).",
	}, []string{"status"})

	// BlocksFound counts valid blocks the pool found and the upstream accepted.
	BlocksFound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oep_blocks_found_total",
		Help: "Valid blocks found by the pool and accepted by the upstream node.",
	})

	// BlockSubmissions counts block submissions to the upstream by result.
	BlockSubmissions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oep_block_submissions_total",
		Help: "Block submissions to the upstream node by result (accepted, rejected, error).",
	}, []string{"result"})

	// StratumSessions tracks the number of currently connected Stratum miners.
	StratumSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oep_stratum_sessions",
		Help: "Currently connected Stratum miners.",
	})

	// UpstreamHealthy reflects each upstream node's health from the latest check.
	UpstreamHealthy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "oep_upstream_healthy",
		Help: "Upstream node health from the latest check (1 = healthy, 0 = sick).",
	}, []string{"url"})

	// BuildInfo is the conventional build-info gauge: its value is always 1 and
	// the running version rides in the "version" label, so dashboards can group
	// and alert on the deployed version.
	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "oep_build_info",
		Help: "Build information; the value is always 1, with the version in the label.",
	}, []string{"version"})
)

// SetBuildInfo records the running version on the build-info gauge.
func SetBuildInfo(version string) {
	BuildInfo.WithLabelValues(version).Set(1)
}

// ShareOutcome records a single share outcome. Use the Share* constants.
func ShareOutcome(status string) {
	ShareOutcomes.WithLabelValues(status).Inc()
}

// SetUpstreamHealthy records an upstream's health as a 0/1 gauge.
func SetUpstreamHealthy(url string, healthy bool) {
	v := 0.0
	if healthy {
		v = 1.0
	}
	UpstreamHealthy.WithLabelValues(url).Set(v)
}
