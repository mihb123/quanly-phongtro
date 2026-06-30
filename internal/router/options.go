package router

import (
	"io/fs"
	"net/netip"
)

type Option func(*options)

type options struct {
	trustedProxies      []netip.Prefix
	uploadURLSigningKey string
	staticFS            fs.FS
}

// WithTrustedProxies configures the proxy CIDRs trusted for X-Forwarded-* headers in DPoP htu.
func WithTrustedProxies(prefixes []netip.Prefix) Option {
	return func(o *options) {
		o.trustedProxies = prefixes
	}
}

// WithUploadURLSigningKey configures HMAC verification for public signed upload URLs.
func WithUploadURLSigningKey(key string) Option {
	return func(o *options) {
		o.uploadURLSigningKey = key
	}
}

// WithStaticFS bật phục vụ frontend SPA đã nhúng từ cùng origin với API.
func WithStaticFS(staticFS fs.FS) Option {
	return func(o *options) {
		o.staticFS = staticFS
	}
}

// newOptions applies router option defaults.
func newOptions(opts ...Option) options {
	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
