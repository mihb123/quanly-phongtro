package router

import (
	"fmt"
	"net/http"
	"net/netip"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
	"go.uber.org/zap"
)

// TraceIDHeader: client có thể gửi trace id; server luôn trả lại ở response header.
const TraceIDHeader = "X-Trace-ID"

const maxTraceIDLen = 64

// validTraceID chỉ nhận [A-Za-z0-9._-] để header lạ không làm bẩn log.
func validTraceID(s string) bool {
	if s == "" || len(s) > maxTraceIDLen {
		return false
	}
	for _, c := range s {
		ok := c == '-' || c == '_' || c == '.' ||
			(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !ok {
			return false
		}
	}
	return true
}

// loggingMiddleware là middleware ngoài cùng: sinh trace_id, resolve client IP,
// gắn logger + request state vào context và log ĐÚNG MỘT dòng cho mỗi request.
// Mức log: 5xx ERROR (kèm stack), 4xx WARN, 2xx/3xx có lỗi/warning ghi nhận WARN, còn lại INFO.
func loggingMiddleware(base *zap.Logger, trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	if base == nil {
		base = zap.L()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			traceID := r.Header.Get(TraceIDHeader)
			if !validTraceID(traceID) {
				traceID = uuid.NewString()
			}
			w.Header().Set(TraceIDHeader, traceID)

			ctx := logging.WithTraceID(r.Context(), base, traceID)
			ctx, state := logging.WithRequestState(ctx)
			clientIP := security.ClientIP(r, trustedProxies)
			if clientIP != "" {
				// Gắn sẵn để handler/logger phía sau không phải quét header lần hai.
				ctx = security.WithClientIP(ctx, clientIP)
			}
			r = r.WithContext(ctx)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK // handler không ghi gì: net/http trả 200
			}
			fields := []zap.Field{
				zap.String("trace_id", traceID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", status),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("latency", time.Since(start)),
				zap.String("ip", clientIP),
			}
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					fields = append(fields, zap.String("route", pattern))
				}
			}

			snap := state.Snapshot()
			err, panicStack := snap.Err, snap.PanicStack
			if snap.UserID != "" {
				fields = append(fields, zap.String("user_id", snap.UserID))
			}
			fields = append(fields, snap.Fields...)
			if len(snap.Warnings) > 0 {
				fields = append(fields, zap.Strings("warnings", snap.Warnings))
			}
			var stack []byte
			if err != nil {
				fields = append(fields, zap.String("error", err.Error()))
				if layer := apperr.FailedLayer(err); layer != "" {
					fields = append(fields, zap.String("failed_layer", layer), zap.String("origin", apperr.Origin(err)))
				}
				stack = apperr.Stack(err)
				if panicStack != nil {
					stack = panicStack
				}
			}

			switch {
			case status >= http.StatusInternalServerError:
				if len(stack) > 0 {
					fields = append(fields, zap.ByteString("stack", stack))
				}
				base.Error("request failed", fields...)
			case status >= http.StatusBadRequest:
				base.Warn("request rejected", fields...)
			case err != nil || len(snap.Warnings) > 0:
				// Response thành công nhưng có lỗi được ghi nhận (vd. webhook luôn trả 200).
				base.Warn("request completed with warnings", fields...)
			default:
				base.Info("request completed", fields...)
			}
		})
	}
}

// recoverMiddleware nằm ngay trong loggingMiddleware: panic -> error + stack tại điểm
// panic ghi vào request state; client chỉ nhận thông điệp chung.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler { //nolint:errorlint // sentinel so sánh trực tiếp theo net/http
				panic(rec)
			}
			err, ok := rec.(error)
			if !ok {
				err = fmt.Errorf("%v", rec)
			}
			logging.RecordPanic(r.Context(), fmt.Errorf("panic: %w", err), debug.Stack())
			writeError(w, http.StatusInternalServerError, httpx.InternalErrorMessage)
		}()
		next.ServeHTTP(w, r)
	})
}
