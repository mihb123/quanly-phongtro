package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

// slowAPILogger trả về middleware đo thời gian xử lý từng request và ghi ra file
// log riêng (logs/slow_api.log) những request vượt ngưỡng SLOW_API_THRESHOLD,
// phục vụ việc theo dõi và tối ưu hiệu năng. Trả về middleware rỗng khi tính năng tắt.
func slowAPILogger(slowLogger *logger.SlowAPILogger) func(http.Handler) http.Handler {
	if slowLogger.Threshold() <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				duration := time.Since(start)
				if !slowLogger.IsSlow(duration) {
					return
				}
				slowLogger.Log(buildSlowAPIEntry(r, ww, duration, start))
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// buildSlowAPIEntry gom thông tin request/response sau khi handler chạy xong.
// Route pattern chỉ có sau khi chi đã match nên phải đọc ở bước này.
func buildSlowAPIEntry(r *http.Request, ww middleware.WrapResponseWriter, duration time.Duration, start time.Time) logger.SlowAPIEntry {
	route := ""
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		route = routeCtx.RoutePattern()
	}

	status := ww.Status()
	if status == 0 {
		status = http.StatusOK
	}

	clientIP := r.RemoteAddr
	if ip, ok := security.ClientIPFromContext(r.Context()); ok {
		clientIP = ip
	}

	return logger.SlowAPIEntry{
		Time:         start,
		Method:       r.Method,
		URI:          r.URL.RequestURI(),
		Path:         r.URL.Path,
		Route:        route,
		QueryParams:  r.URL.RawQuery,
		StatusCode:   status,
		Duration:     duration,
		ClientIP:     clientIP,
		UserAgent:    r.UserAgent(),
		RequestSize:  r.ContentLength,
		ResponseSize: ww.BytesWritten(),
	}
}
