package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/types"
)

// BUGS #25: a handler could not learn the client's address (the Request had no peer address),
// so per-address limits in identity were one global bucket.
func TestClientIP(t *testing.T) {
	trusted := types.ParsePrefixes("10.0.0.0/8, 127.0.0.1, bogus")
	cases := []struct {
		name, peer, xff, realIP, want string
	}{
		{"direct client", "203.0.113.7:5000", "", "", "203.0.113.7"},
		{"untrusted peer cannot choose its address", "203.0.113.7:5000", "198.51.100.1", "198.51.100.2", "203.0.113.7"},
		{"behind a trusted proxy", "10.0.0.2:443", "198.51.100.9", "", "198.51.100.9"},
		{"rightmost untrusted hop wins over a spoofed left one", "10.0.0.2:443", "1.2.3.4, 198.51.100.9, 10.0.0.3", "", "198.51.100.9"},
		{"every hop trusted: the leftmost", "127.0.0.1:80", "10.1.1.1, 10.0.0.3", "", "10.1.1.1"},
		{"X-Real-Ip from a trusted proxy", "127.0.0.1:80", "", "198.51.100.4", "198.51.100.4"},
		{"IPv6 peer", "[2001:db8::1]:8080", "", "", "2001:db8::1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.peer
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if c.realIP != "" {
			r.Header.Set("X-Real-Ip", c.realIP)
		}
		req := api.NewRequest(r)
		if got := types.ClientIP(req, trusted); got != c.want {
			t.Errorf("%s: ClientIP = %q, want %q", c.name, got, c.want)
		}
	}
	if len(trusted) != 2 {
		t.Errorf("ParsePrefixes kept %d prefixes, want 2 (a malformed entry is skipped)", len(trusted))
	}
	if types.RemoteAddr(nil) != "" || types.ClientIP(nil, trusted) != "" {
		t.Error("no request, no address")
	}
}
