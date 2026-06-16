package router

import "strings"

type Option func(*options)

type options struct {
	dpopVerificationURL string
	uploadURLSigningKey string
}

// WithDPoPVerificationURL configures the external API base URL used for DPoP htu.
func WithDPoPVerificationURL(baseURL string) Option {
	return func(o *options) {
		o.dpopVerificationURL = strings.TrimRight(baseURL, "/")
	}
}

// WithUploadURLSigningKey configures HMAC verification for public signed upload URLs.
func WithUploadURLSigningKey(key string) Option {
	return func(o *options) {
		o.uploadURLSigningKey = key
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
