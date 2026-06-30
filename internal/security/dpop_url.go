package security

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// BuildDPoPHTU builds the absolute URI used for DPoP htu verification.
// It mirrors the origin the browser actually used so it matches the frontend
// proof (which signs window.location.origin). Scheme/host come from the real
// request; X-Forwarded-Proto/Host are honored only when the immediate peer is a
// configured trusted proxy (e.g. a TLS-terminating tunnel in front of the app).
func BuildDPoPHTU(r *http.Request, trustedProxies []netip.Prefix) string {
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	trusted := fromTrustedProxy(r, trustedProxies)
	return requestScheme(r, trusted) + "://" + requestHost(r, trusted) + path
}

// requestScheme resolves the public scheme, preferring X-Forwarded-Proto from a trusted proxy.
func requestScheme(r *http.Request, trustedPeer bool) string {
	if trustedPeer {
		if proto := forwardedFirst(r.Header.Get("X-Forwarded-Proto")); proto != "" {
			return strings.ToLower(proto)
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// requestHost resolves the public host, preferring X-Forwarded-Host from a trusted proxy.
func requestHost(r *http.Request, trustedPeer bool) string {
	if trustedPeer {
		if host := forwardedFirst(r.Header.Get("X-Forwarded-Host")); host != "" {
			return host
		}
	}
	return r.Host
}

// fromTrustedProxy reports whether the immediate peer belongs to a trusted proxy CIDR.
func fromTrustedProxy(r *http.Request, trustedProxies []netip.Prefix) bool {
	if len(trustedProxies) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	for _, prefix := range trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// forwardedFirst returns the first entry of a comma-separated forwarded header.
func forwardedFirst(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}
