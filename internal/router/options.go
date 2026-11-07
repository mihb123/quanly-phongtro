package router

import (
	"io/fs"
	"net/netip"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/logging/filelog"
	"go.uber.org/zap"
)

type Option func(*options)

type options struct {
	trustedProxies      []netip.Prefix
	uploadURLSigningKey string
	staticFS            fs.FS
	slowAPILogger       *filelog.SlowAPILogger
	securityLogger      *filelog.SecurityLogger
	pluginRoutes        func(chi.Router)
	logger              *zap.Logger
}

// WithLogger đặt logger gốc cho dòng log mỗi request (mặc định zap.L()).
func WithLogger(l *zap.Logger) Option {
	return func(o *options) {
		o.logger = l
	}
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

// WithSlowAPILogger bật ghi log các request vượt ngưỡng chậm ra file riêng.
func WithSlowAPILogger(slowLogger *filelog.SlowAPILogger) Option {
	return func(o *options) {
		o.slowAPILogger = slowLogger
	}
}

func WithSecurityLogger(securityLogger *filelog.SecurityLogger) Option {
	return func(o *options) {
		o.securityLogger = securityLogger
	}
}

func WithPluginRoutes(routes func(chi.Router)) Option {
	return func(o *options) {
		o.pluginRoutes = routes
	}
}

func WithPluginPublicRoutes(routes func(chi.Router)) Option {
	return func(o *options) {
		o.pluginPublicRoutes = routes
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
