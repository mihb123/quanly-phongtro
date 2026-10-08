package zalobot

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
	"syscall"
	"time"
)

const (
	DefaultImageMaxBytes = 5 << 20
	defaultImageTimeout  = 10 * time.Second
	maxImageRedirects    = 3
)

type ImageDownloader struct {
	Transport http.RoundTripper
	LookupIP  func(ctx context.Context, host string) ([]netip.Addr, error)
	Timeout   time.Duration
	MaxBytes  int64
}

func DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	return (&ImageDownloader{}).Download(ctx, imageURL)
}

// Download downloads and validates an HTTPS image sent to the bot.
func (d *ImageDownloader) Download(ctx context.Context, imageURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout())
	defer cancel()

	if err := d.ValidateURL(ctx, imageURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create image download request: %w", err)
	}

	resp, err := d.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	maxBytes := d.maxBytes()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrInvalidImage, resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("%w: image exceeds maximum size", ErrInvalidImage)
	}
	if !isAllowedImageContentType(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: unsupported image content type", ErrInvalidImage)
	}

	imageBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if int64(len(imageBytes)) > maxBytes {
		return nil, fmt.Errorf("%w: image exceeds maximum size", ErrInvalidImage)
	}

	return imageBytes, nil
}

// ValidateURL verifies scheme and resolved network targets.
func (d *ImageDownloader) ValidateURL(ctx context.Context, rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidImageURL, err)
	}
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: must use https", ErrInvalidImageURL)
	}
	if parsedURL.Hostname() == "" {
		return fmt.Errorf("%w: host is required", ErrInvalidImageURL)
	}

	addresses, err := d.lookupIP(ctx, parsedURL.Hostname())
	if err != nil {
		return err
	}
	for _, addr := range addresses {
		if !isPublicRoutableAddr(addr) {
			return fmt.Errorf("%w: resolves to disallowed address", ErrInvalidImageURL)
		}
	}

	return nil
}

// httpClient returns a timeout-bound client with guarded redirects.
func (d *ImageDownloader) httpClient() *http.Client {
	transport := d.Transport
	if transport == nil {
		transport = publicOnlyTransport()
	}
	return &http.Client{
		Timeout:   d.timeout(),
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxImageRedirects {
				return errors.New("too many redirects")
			}
			if len(via) > 0 && !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
				return errors.New("redirect to different host is not allowed")
			}
			return d.ValidateURL(req.Context(), req.URL.String())
		},
	}
}

func (d *ImageDownloader) timeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return defaultImageTimeout
}

func (d *ImageDownloader) maxBytes() int64 {
	if d.MaxBytes > 0 {
		return d.MaxBytes
	}
	return DefaultImageMaxBytes
}

func (d *ImageDownloader) lookupIP(ctx context.Context, host string) ([]netip.Addr, error) {
	if d.LookupIP != nil {
		return d.LookupIP(ctx, host)
	}
	return defaultLookupIP(ctx, host)
}

func publicOnlyTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout: defaultImageTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			addr, err := netip.ParseAddr(host)
			if err != nil || !isPublicRoutableAddr(addr) {
				return fmt.Errorf("%w: resolves to disallowed address", ErrInvalidImageURL)
			}
			return nil
		},
	}
	return &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: defaultImageTimeout,
		ForceAttemptHTTP2:   true,
	}
}

// defaultLookupIP resolves a host or parses an IP literal for SSRF checks.
func defaultLookupIP(ctx context.Context, host string) ([]netip.Addr, error) {
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
	addr = addr.Unmap()
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
