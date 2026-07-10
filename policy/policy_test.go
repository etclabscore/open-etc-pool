package policy

import (
	"testing"

	"github.com/etclabscore/open-etc-pool/util"
)

func TestIsBannedDoesNotAllocate(t *testing.T) {
	s := &PolicyServer{config: &Config{}, stats: make(map[string]*Stats)}
	if s.IsBanned("1.2.3.4") {
		t.Fatal("an unknown IP must not be reported as banned")
	}
	if len(s.stats) != 0 {
		t.Fatalf("IsBanned allocated a stats entry for an unknown IP (%d entries)", len(s.stats))
	}
}

func TestEvictIdleStats(t *testing.T) {
	now := util.MakeTimestamp()
	s := &PolicyServer{config: &Config{}, stats: make(map[string]*Stats), timeout: 1000}
	s.stats["idle"] = &Stats{LastBeat: now - 5000}             // idle past the window
	s.stats["fresh"] = &Stats{LastBeat: now}                   // recently active
	s.stats["banned"] = &Stats{LastBeat: now - 5000, Banned: 1} // idle but banned

	s.evictIdleLocked()

	if _, ok := s.stats["idle"]; ok {
		t.Error("idle entry should have been evicted")
	}
	if _, ok := s.stats["fresh"]; !ok {
		t.Error("fresh entry should have been kept")
	}
	if _, ok := s.stats["banned"]; !ok {
		t.Error("banned entry should be kept even when idle")
	}
}

// newTestServer builds a PolicyServer without Start() so the share-policy
// decision can be exercised without a Redis backend or the background workers.
func newTestServer(cfg *Config) *PolicyServer {
	return &PolicyServer{config: cfg, stats: make(map[string]*Stats)}
}

// With banning disabled, ApplySharePolicy must accept every share — even under
// the degenerate CheckThreshold=0/InvalidPercent=0 config, where the ratio
// branch would otherwise reject the very first valid share (ratio 0 >= 0).
func TestApplySharePolicyBanningDisabledAcceptsShares(t *testing.T) {
	s := newTestServer(&Config{
		Banning: Banning{Enabled: false, CheckThreshold: 0, InvalidPercent: 0},
	})
	ip := "10.0.0.1"

	if !s.ApplySharePolicy(ip, true) {
		t.Fatal("valid share rejected with banning disabled")
	}
	if !s.ApplySharePolicy(ip, false) {
		t.Fatal("invalid share rejected with banning disabled (nothing to ban)")
	}
	if s.IsBanned(ip) {
		t.Fatal("client banned with banning disabled")
	}
}

// With banning enabled, a high invalid-share ratio past the check threshold must
// still reject the share and ban the client — the disabled-guard must not have
// disturbed this path.
func TestApplySharePolicyBansOnHighInvalidRatio(t *testing.T) {
	s := newTestServer(&Config{
		Banning: Banning{Enabled: true, CheckThreshold: 2, InvalidPercent: 50},
	})
	ip := "10.0.0.2"

	if !s.ApplySharePolicy(ip, true) {
		t.Fatal("first share below threshold should be accepted")
	}
	// Second share crosses the threshold with a 1:1 invalid ratio (100% >= 50%).
	if s.ApplySharePolicy(ip, false) {
		t.Fatal("share above the invalid-ratio threshold should be rejected")
	}
	if !s.IsBanned(ip) {
		t.Fatal("client should be banned after crossing the invalid-ratio threshold")
	}
}

// The banning-disabled guard must sit after the counter block so the Limits
// reward still applies: a valid share with Limits on / banning off must grow the
// connection allowance by LimitJump. This pins the placement — hoisting the
// guard to the top of ApplySharePolicy would skip incrLimit and fail here.
func TestApplySharePolicyValidShareRewardsLimitWhenBanningDisabled(t *testing.T) {
	s := newTestServer(&Config{
		Banning: Banning{Enabled: false},
		Limits:  Limits{Enabled: true, Limit: 10, LimitJump: 5},
	})
	ip := "10.0.0.3"

	x := s.Get(ip)
	if x.ConnLimit != 10 {
		t.Fatalf("initial ConnLimit = %d, want 10 (Limits.Limit)", x.ConnLimit)
	}
	if !s.ApplySharePolicy(ip, true) {
		t.Fatal("valid share rejected with banning disabled")
	}
	if x.ConnLimit != 15 {
		t.Fatalf("ConnLimit = %d after a valid share, want 15 (10 + LimitJump)", x.ConnLimit)
	}
}
