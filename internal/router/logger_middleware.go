package router

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// realIPLogFormatter wraps chi's DefaultLogFormatter to log the real client IP
// and scheme (https when forwarded via trusted proxies).
type realIPLogFormatter struct {
	formatter      middleware.LogFormatter
	trustedProxies []netip.Prefix
}

func newRealIPLogFormatter(trustedProxies []netip.Prefix, out io.Writer) middleware.LogFormatter {
	if out == nil {
		out = os.Stdout
	}
	return &realIPLogFormatter{
		formatter: &middleware.DefaultLogFormatter{
			Logger:  log.New(out, "", log.LstdFlags),
			NoColor: runtime.GOOS == "windows",
		},
		trustedProxies: trustedProxies,
	}
}

func (f *realIPLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	// Bản copy chỉ sống trong phạm vi hàm này: DefaultLogFormatter đọc field rồi
	// dựng sẵn chuỗi log, không giữ lại con trỏ request, nên request gốc không bị ảnh hưởng.
	rCopy := *r

	// RemoteAddr là địa chỉ của proxy nội bộ, thay bằng IP thật của client.
	// Bỏ port vì đó là port kết nối của proxy, không phải của client.
	if clientIP := security.ResolvedClientIP(r, f.trustedProxies); clientIP != "" {
		rCopy.RemoteAddr = clientIP
	}

	// Nếu request đến từ trusted proxy và được forward với https (qua X-Forwarded-Proto
	// hoặc CF-Visitor của Cloudflare) thì gán TLS giả để formatter in scheme https.
	if r.TLS == nil && security.FromTrustedProxy(r, f.trustedProxies) {
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "" && strings.Contains(r.Header.Get("CF-Visitor"), "https") {
			proto = "https"
		}
		if strings.EqualFold(proto, "https") {
			rCopy.TLS = &tls.ConnectionState{}
		}
	}

	return f.formatter.NewLogEntry(&rCopy)
}

// requestLogger returns a Chi middleware that logs each request using Chi's
// standard request logger formatting, but with the real client IP (extracted safely
// from trusted proxies like Cloudflare Tunnel, Nginx, or AWS ALB) and accurate
// scheme (https when TLS is terminated at the proxy).
// It also injects the resolved client IP into the request context so downstream
// handlers and loggers (internal/service/logger) can access it directly.
func requestLogger(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return customRequestLogger(trustedProxies, nil)
}

func customRequestLogger(trustedProxies []netip.Prefix, out io.Writer) func(http.Handler) http.Handler {
	baseLogger := middleware.RequestLogger(newRealIPLogFormatter(trustedProxies, out))

	return func(next http.Handler) http.Handler {
		logged := baseLogger(next)

		// Gắn client IP vào context *trước* baseLogger để formatter dùng lại giá trị
		// đã resolve thay vì quét header lần hai, và để log luôn khớp với giá trị handler thấy.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if clientIP := security.ClientIP(r, trustedProxies); clientIP != "" {
				r = r.WithContext(security.WithClientIP(r.Context(), clientIP))
			}
			logged.ServeHTTP(w, r)
		})
	}
}
