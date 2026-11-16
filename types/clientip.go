package types

import (
	"net"
	"net/netip"
	"strings"
)

// RemoteAddresser is implemented by the requests of the HTTP platforms (api, web and spa): the
// connection's peer address. It is optional so other Request implementations (cli, test fakes)
// keep compiling; RemoteAddr reports "" for them.
type RemoteAddresser interface {
	RemoteAddr() string
}

// RemoteAddr is the request's peer address ("host:port"), or "" when the platform has none.
func RemoteAddr(req Request) string {
	if r, ok := req.(RemoteAddresser); ok {
		return r.RemoteAddr()
	}
	return ""
}

// ClientIP is the address of the client behind a request. The peer address is the answer
// unless the peer is one of the trusted proxies; then X-Forwarded-For is read from the right,
// skipping trusted proxies, and the first address that is not one is the client (the leftmost
// entry when every hop is trusted), with X-Real-Ip as the fallback. Forwarded headers from an
// untrusted peer are ignored, so a client cannot choose its own address. "" when the request has
// no peer address.
func ClientIP(req Request, trusted []netip.Prefix) string {
	if req == nil {
		return ""
	}
	peer, ok := parseAddr(RemoteAddr(req))
	if !ok {
		return ""
	}
	if !inAny(peer, trusted) {
		return peer.String()
	}
	headers := req.Headers()
	if xff := header(headers, "X-Forwarded-For"); xff != "" {
		hops := strings.Split(xff, ",")
		client := ""
		for i := len(hops) - 1; i >= 0; i-- {
			a, ok := parseAddr(strings.TrimSpace(hops[i]))
			if !ok {
				break // a malformed hop: trust nothing to its left
			}
			client = a.String()
			if !inAny(a, trusted) {
				return client
			}
		}
		if client != "" {
			return client
		}
	}
	if a, ok := parseAddr(strings.TrimSpace(header(headers, "X-Real-Ip"))); ok {
		return a.String()
	}
	return peer.String()
}

// ParsePrefixes reads a comma-separated list of CIDRs or single addresses ("10.0.0.0/8,
// 127.0.0.1"), skipping anything malformed.
func ParsePrefixes(list string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range strings.Split(list, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if p, err := netip.ParsePrefix(s); err == nil {
			out = append(out, p.Masked())
			continue
		}
		if a, err := netip.ParseAddr(s); err == nil {
			out = append(out, netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()))
		}
	}
	return out
}

func parseAddr(s string) (netip.Addr, bool) {
	if s == "" {
		return netip.Addr{}, false
	}
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	a, err := netip.ParseAddr(strings.Trim(s, "[]"))
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap().WithZone(""), true
}

func inAny(a netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// header reads a header case-insensitively (net/http stores canonical keys, test fakes may not).
func header(headers map[string][]string, name string) string {
	if vs, ok := headers[name]; ok && len(vs) > 0 {
		return vs[0]
	}
	for k, vs := range headers {
		if strings.EqualFold(k, name) && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}
