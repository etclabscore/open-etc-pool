package proxy

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// writeError must set the Content-Type before WriteHeader (so it isn't dropped)
// and write the message to the body.
func TestWriteError(t *testing.T) {
	s := &ProxyServer{}
	rec := httptest.NewRecorder()

	s.writeError(rec, 405, "rpc: POST method required")

	if rec.Code != 405 {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, "POST method required") {
		t.Errorf("body = %q, must contain the error message", body)
	}
}
