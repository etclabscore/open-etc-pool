package proxy

import (
	"net/http"
	"testing"
)

func newReq(remoteAddr, xff string) *http.Request {
	r := &http.Request{RemoteAddr: remoteAddr, Header: http.Header{}}
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestRemoteAddr(t *testing.T) {
	direct := &ProxyServer{config: &Config{Proxy: Proxy{BehindReverseProxy: false}}}
	behind := &ProxyServer{config: &Config{Proxy: Proxy{BehindReverseProxy: true}}}

	cases := []struct {
		name   string
		server *ProxyServer
		remote string
		xff    string
		want   string
	}{
		{"not behind proxy ignores XFF", direct, "203.0.113.7:5000", "1.1.1.1", "203.0.113.7"},
		{"behind proxy, no XFF, uses peer", behind, "203.0.113.7:5000", "", "203.0.113.7"},
		{"behind proxy uses rightmost hop", behind, "10.0.0.1:5000", "1.1.1.1, 2.2.2.2, 9.9.9.9", "9.9.9.9"},
		// The client prepends a spoofed IP; our proxy appends the real peer last.
		{"spoofed left entry is ignored", behind, "10.0.0.1:5000", "6.6.6.6, 9.9.9.9", "9.9.9.9"},
		{"whitespace is trimmed", behind, "10.0.0.1:5000", "evil , 9.9.9.9", "9.9.9.9"},
		{"single valid entry", behind, "10.0.0.1:5000", "9.9.9.9", "9.9.9.9"},
		{"invalid rightmost falls back to peer", behind, "203.0.113.7:5000", "9.9.9.9, garbage", "203.0.113.7"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.server.remoteAddr(newReq(c.remote, c.xff)); got != c.want {
				t.Fatalf("remoteAddr(%q, XFF=%q) = %q, want %q", c.remote, c.xff, got, c.want)
			}
		})
	}
}
