package proxy

import (
	"sync"
	"testing"

	"github.com/etclabscore/go-etchash"
)

// getHasher must be safe under concurrent first use — the previous code did an
// unsynchronized nil-check + assignment on a package global. Run with -race.
func TestGetHasherConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	got := make([]*etchash.Etchash, 16)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i] = getHasher("classic")
		}(i)
	}
	wg.Wait()

	for i, h := range got {
		if h == nil {
			t.Fatalf("goroutine %d got a nil hasher", i)
		}
		if h != got[0] {
			t.Fatalf("goroutine %d got a different hasher instance", i)
		}
	}
}
