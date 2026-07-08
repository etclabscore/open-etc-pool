package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/etclabscore/open-etc-pool/storage"
)

const testAddr = "0x0000000000000000000000000000000000000000"

func newTestServer(endpoint string) *ApiServer {
	backend := storage.NewRedisClient(&storage.Config{Endpoint: endpoint}, "test-api")
	return NewApiServer(&ApiConfig{
		StatsCollectInterval: "5s",
		HashrateWindow:       "30m",
		HashrateLargeWindow:  "3h",
		Payments:             30,
	}, backend)
}

func accountRequest(login string) *http.Request {
	req := httptest.NewRequest("GET", "/api/accounts/"+login, nil)
	return mux.SetURLVars(req, map[string]string{"login": login})
}

// A backend error must surface as 500, not be masked as a 404.
func TestAccountIndexBackendError(t *testing.T) {
	s := newTestServer("127.0.0.1:1") // unreachable Redis -> IsMinerExists errors
	rec := httptest.NewRecorder()
	s.AccountIndex(rec, accountRequest(testAddr))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (a backend error must not become 404)", rec.Code)
	}
}

// A miner that does not exist, with a healthy backend, is a genuine 404.
func TestAccountIndexNotFound(t *testing.T) {
	s := newTestServer("127.0.0.1:6379")
	rec := httptest.NewRecorder()
	s.AccountIndex(rec, accountRequest(testAddr))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
