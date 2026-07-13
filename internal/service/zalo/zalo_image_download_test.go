package zalo

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type imageRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip lets tests stub image download responses.
func (f imageRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestValidateZaloTransactionImageURLRejectsUnsafeTargets covers SSRF URL validation.
func TestValidateZaloTransactionImageURLRejectsUnsafeTargets(t *testing.T) {
	ctx := context.Background()

	if err := validateZaloTransactionImageURL(ctx, "http://zalo.test/image.jpg"); err == nil {
		t.Fatal("expected non-https URL to be rejected")
	}

	previousResolver := resolveZaloImageHost
	resolveZaloImageHost = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	defer func() { resolveZaloImageHost = previousResolver }()

	if err := validateZaloTransactionImageURL(ctx, "https://zalo.test/image.jpg"); err == nil {
		t.Fatal("expected loopback resolved URL to be rejected")
	}
}

// TestDownloadZaloTransactionImageRejectsBadResponses covers content type and size limits.
func TestDownloadZaloTransactionImageRejectsBadResponses(t *testing.T) {
	ctx := context.Background()

	previousResolver := resolveZaloImageHost
	resolveZaloImageHost = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	defer func() { resolveZaloImageHost = previousResolver }()

	tests := []struct {
		name          string
		contentType   string
		contentLength int64
		body          string
	}{
		{
			name:          "non image content type",
			contentType:   "text/plain",
			contentLength: 4,
			body:          "text",
		},
		{
			name:          "too large content length",
			contentType:   "image/jpeg",
			contentLength: maxZaloTransactionImageBytes + 1,
			body:          "img",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousClientFactory := newZaloImageHTTPClient
			newZaloImageHTTPClient = func() *http.Client {
				return &http.Client{Transport: imageRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode:    http.StatusOK,
						Header:        http.Header{"Content-Type": []string{tt.contentType}},
						ContentLength: tt.contentLength,
						Body:          io.NopCloser(strings.NewReader(tt.body)),
						Request:       req,
					}, nil
				})}
			}
			defer func() { newZaloImageHTTPClient = previousClientFactory }()

			if _, err := downloadZaloTransactionImage(ctx, "https://zalo.test/image.jpg"); err == nil {
				t.Fatal("expected download to fail")
			}
		})
	}
}
