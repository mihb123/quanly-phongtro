package security

import (
	"net/http"
	"net/url"
	"strings"
)

// BuildDPoPHTU builds the absolute URI used for DPoP htu verification.
func BuildDPoPHTU(r *http.Request, baseURL string) string {
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	if baseURL != "" {
		base, err := url.Parse(strings.TrimRight(baseURL, "/"))
		if err == nil && base.Scheme != "" && base.Host != "" {
			return base.Scheme + "://" + base.Host + path
		}
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if r.URL.Scheme != "" && r.URL.Host != "" {
		scheme = r.URL.Scheme
		host = r.URL.Host
	}

	return scheme + "://" + host + path
}
