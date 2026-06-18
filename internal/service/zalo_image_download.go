package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

const maxZaloTransactionImageBytes = 5 << 20

var (
	newZaloImageHTTPClient = defaultZaloImageHTTPClient
	resolveZaloImageHost   = defaultResolveURLHost
)

// downloadZaloTransactionImage downloads and validates an HTTPS image from Zalo.
func downloadZaloTransactionImage(ctx context.Context, imageURL string) ([]byte, error) {
	if err := validateZaloTransactionImageURL(ctx, imageURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create image download request: %w", err)
	}

	resp, err := newZaloImageHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download image status: %d", resp.StatusCode)
	}
	if resp.ContentLength > maxZaloTransactionImageBytes {
		return nil, fmt.Errorf("image exceeds maximum size")
	}
	if !isAllowedImageContentType(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("unsupported image content type")
	}

	limitedBody := io.LimitReader(resp.Body, maxZaloTransactionImageBytes+1)
	imageBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(imageBytes) > maxZaloTransactionImageBytes {
		return nil, fmt.Errorf("image exceeds maximum size")
	}

	return imageBytes, nil
}

// defaultZaloImageHTTPClient returns a timeout-bound client with guarded redirects.
func defaultZaloImageHTTPClient() *http.Client {
	return &http.Client{
		Timeout: zaloTransactionImageDownloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if len(via) > 0 && !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
				return errors.New("redirect to different host is not allowed")
			}
			return validateZaloTransactionImageURL(req.Context(), req.URL.String())
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
}

// validateZaloTransactionImageURL verifies scheme and resolved network targets.
func validateZaloTransactionImageURL(ctx context.Context, rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse image url: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return errors.New("image url must use https")
	}
	if parsedURL.Hostname() == "" {
		return errors.New("image url host is required")
	}

	addresses, err := resolveZaloImageHost(ctx, parsedURL.Hostname())
	if err != nil {
		return err
	}
	for _, addr := range addresses {
		if !isPublicRoutableAddr(addr) {
			return fmt.Errorf("image url resolves to disallowed address")
		}
	}

	return nil
}

// defaultResolveURLHost resolves a host or parses an IP literal for SSRF checks.
func defaultResolveURLHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr}, nil
	}

	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve image url host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("image url host has no addresses")
	}

	return addresses, nil
}

// isPublicRoutableAddr rejects internal, local, multicast, and unspecified targets.
func isPublicRoutableAddr(addr netip.Addr) bool {
	return addr.IsValid() &&
		addr.IsGlobalUnicast() &&
		!addr.IsPrivate() &&
		!addr.IsLoopback() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsLinkLocalMulticast() &&
		!addr.IsMulticast() &&
		!addr.IsUnspecified()
}

// isAllowedImageContentType accepts common raster image response types.
func isAllowedImageContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
