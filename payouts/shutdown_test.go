package payouts

import (
	"context"
	"testing"
	"time"

	"github.com/etclabscore/open-etc-pool/storage"
)

// Graceful shutdown: the block unlocker and payouts processor must return from
// Start when their context is cancelled, instead of looping forever. The daemon
// is pointed at an unreachable address so the initial cycle fails fast (its
// errors are handled) and we reach the select loop, where the cancelled context
// must make Start return promptly.

func assertStartReturns(t *testing.T, name string, start func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		start()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("%s.Start did not return after its context was cancelled", name)
	}
}

func TestBlockUnlockerStopsOnContextCancel(t *testing.T) {
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, "test-graceful-unlocker")
	network := "classic"
	u := NewBlockUnlocker(&UnlockerConfig{
		Interval:      "1h",
		Depth:         120,
		ImmatureDepth: 20,
		Daemon:        "http://127.0.0.1:1",
		Timeout:       "1s",
	}, backend, &network)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assertStartReturns(t, "BlockUnlocker", func() { u.Start(ctx) })
}

func TestPayoutsProcessorStopsOnContextCancel(t *testing.T) {
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, "test-graceful-payer")
	u := NewPayoutsProcessor(&PayoutsConfig{
		Interval: "1h",
		Daemon:   "http://127.0.0.1:1",
		Timeout:  "1s",
	}, backend)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assertStartReturns(t, "PayoutsProcessor", func() { u.Start(ctx) })
}
