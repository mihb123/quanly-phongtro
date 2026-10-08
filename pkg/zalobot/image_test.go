package zalobot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type imageRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip lets tests stub image download responses.
func (f imageRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func lookupFixed(addr string) func(context.Context, string) ([]netip.Addr, error) {
	return func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr(addr)}, nil
	}
}

func imageResponse(contentType string, contentLength int64, body string) http.RoundTripper {
	return imageRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": []string{contentType}},
			ContentLength: contentLength,
			Body:          io.NopCloser(strings.NewReader(body)),
			Request:       req,
		}, nil
	})
}

func TestImageDownloaderValidateURLRejectsUnsafeTargets(t *testing.T) {
	ctx := context.Background()
	downloader := &ImageDownloader{LookupIP: lookupFixed("8.8.8.8")}

	if err := downloader.ValidateURL(ctx, "http://zalo.test/image.jpg"); !errors.Is(err, ErrInvalidImageURL) {
		t.Fatalf("expected non-https URL to be rejected, got %v", err)
	}

	for _, addr := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "::ffff:127.0.0.1"} {
		downloader.LookupIP = lookupFixed(addr)
		if err := downloader.ValidateURL(ctx, "https://zalo.test/image.jpg"); !errors.Is(err, ErrInvalidImageURL) {
			t.Fatalf("expected %s to be rejected, got %v", addr, err)
		}
	}
}

func TestImageDownloaderRejectsBadResponses(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		contentType   string
		contentLength int64
		body          string
		maxBytes      int64
	}{
		{name: "non image content type", contentType: "text/plain", contentLength: 4, body: "text"},
		{name: "too large content length", contentType: "image/jpeg", contentLength: DefaultImageMaxBytes + 1, body: "img"},
		{name: "too large body", contentType: "image/png", contentLength: -1, body: "123456", maxBytes: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &ImageDownloader{
				Transport: imageResponse(tt.contentType, tt.contentLength, tt.body),
				LookupIP:  lookupFixed("8.8.8.8"),
				MaxBytes:  tt.maxBytes,
			}
			if _, err := downloader.Download(ctx, "https://zalo.test/image.jpg"); !errors.Is(err, ErrInvalidImage) {
				t.Fatalf("expected invalid image error, got %v", err)
			}
		})
	}
}

func TestImageDownloaderDownload(t *testing.T) {
	downloader := &ImageDownloader{
		Transport: imageResponse("image/jpeg; charset=binary", 3, "img"),
		LookupIP:  lookupFixed("8.8.8.8"),
	}

	got, err := downloader.Download(context.Background(), "https://zalo.test/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "img" {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestImageDownloaderRejectsCrossHostRedirect(t *testing.T) {
	downloader := &ImageDownloader{
		Transport: imageRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://other.test/image.jpg"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}),
		LookupIP: lookupFixed("8.8.8.8"),
	}

	if _, err := downloader.Download(context.Background(), "https://zalo.test/image.jpg"); err == nil {
		t.Fatal("expected cross-host redirect to fail")
	}
}

func TestPublicOnlyTransportRejectsPrivateDial(t *testing.T) {
	_, err := publicOnlyTransport().DialContext(context.Background(), "tcp", "127.0.0.1:443")
	if !errors.Is(err, ErrInvalidImageURL) {
		t.Fatalf("expected private dial to be rejected, got %v", err)
	}
}

// TestIsAllowedImageContentType covers accepted, rejected, and malformed content types.
func TestIsAllowedImageContentType(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "image/png; charset=utf-8", "IMAGE/GIF", "image/webp"} {
		if !isAllowedImageContentType(ct) {
			t.Errorf("expected %q allowed", ct)
		}
	}
	for _, ct := range []string{"text/plain", "application/json", "", "!!!bad"} {
		if isAllowedImageContentType(ct) {
			t.Errorf("expected %q rejected", ct)
		}
	}
}

// TestDefaultLookupIP covers IP literal parsing.
func TestDefaultLookupIP(t *testing.T) {
	addrs, err := defaultLookupIP(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || addrs[0].String() != "8.8.8.8" {
		t.Errorf("unexpected addresses: %v", addrs)
	}

	addrs, err = defaultLookupIP(context.Background(), "localhost")
	if err != nil {
		t.Fatalf("unexpected localhost resolve error: %v", err)
	}
	if len(addrs) == 0 {
		t.Error("expected localhost to resolve at least one address")
	}

	if _, err := defaultLookupIP(context.Background(), "invalid host name"); err == nil {
		t.Error("expected resolver error for malformed host")
	}
}

// TestImageDownloaderValidateURL_Errors covers parse, scheme, host, and resolver failures.
func TestImageDownloaderValidateURL_Errors(t *testing.T) {
	ctx := context.Background()
	downloader := &ImageDownloader{LookupIP: func(context.Context, string) ([]netip.Addr, error) {
		return nil, errors.New("resolve failed")
	}}

	if err := downloader.ValidateURL(ctx, "https://%zz"); err == nil {
		t.Error("expected parse error")
	}
	if err := downloader.ValidateURL(ctx, "http://host/x"); err == nil {
		t.Error("expected scheme error")
	}
	if err := downloader.ValidateURL(ctx, "https:///path"); err == nil {
		t.Error("expected empty host error")
	}

	if err := downloader.ValidateURL(ctx, "https://zalo.test/x.jpg"); err == nil {
		t.Error("expected resolver error")
	}
}

// TestImageDownloaderValidateURL_PublicAllowed accepts a public routable target.
func TestImageDownloaderValidateURL_PublicAllowed(t *testing.T) {
	downloader := &ImageDownloader{LookupIP: func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}}
	if err := downloader.ValidateURL(context.Background(), "https://zalo.test/x.jpg"); err != nil {
		t.Errorf("expected public target allowed, got %v", err)
	}
}

// TestImageDownloaderHTTPClient exercises the redirect guard branches.
func TestImageDownloaderHTTPClient(t *testing.T) {
	downloader := &ImageDownloader{}
	client := downloader.httpClient()
	if client.Timeout != defaultImageTimeout {
		t.Errorf("unexpected timeout: %v", client.Timeout)
	}

	mustReq := func(rawURL string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		return req
	}

	// Too many redirects.
	via := []*http.Request{mustReq("https://a.test/1"), mustReq("https://a.test/2"), mustReq("https://a.test/3")}
	if err := client.CheckRedirect(mustReq("https://a.test/4"), via); err == nil {
		t.Error("expected too-many-redirects error")
	}

	// Redirect to a different host.
	via = []*http.Request{mustReq("https://a.test/1")}
	if err := client.CheckRedirect(mustReq("https://b.test/2"), via); err == nil {
		t.Error("expected cross-host redirect error")
	}

	// Same-host redirect that passes host validation via stubbed resolver.
	downloader.LookupIP = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	via = []*http.Request{mustReq("https://a.test/1")}
	if err := client.CheckRedirect(mustReq("https://a.test/2"), via); err != nil {
		t.Errorf("expected allowed same-host redirect, got %v", err)
	}
}

func TestImageDownloaderTimeoutCoversLookup(t *testing.T) {
	downloader := &ImageDownloader{
		Timeout: 10 * time.Millisecond,
		LookupIP: func(ctx context.Context, _ string) ([]netip.Addr, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			}
		},
		Transport: imageResponse("image/jpeg", 3, "img"),
	}

	start := time.Now()
	_, err := downloader.Download(context.Background(), "https://zalo.test/image.jpg")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("timeout did not cover lookup, took %v", elapsed)
	}
}
