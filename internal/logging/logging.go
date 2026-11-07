// Package logging khởi tạo zap và truyền logger/trace_id qua context.
// Đặt ở internal/ (không phải pkg/) vì phụ thuộc go.uber.org/zap.
package logging

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}
type traceKey struct{}

// New: console khi env là "dev"/"development", còn lại (mặc định "prod") là JSON.
func New(env, level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	if IsDev(env) {
		cfg = zap.NewDevelopmentConfig()
	}
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.DisableStacktrace = true // stack do apperr cung cấp, không phải zap chụp tại điểm log
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err == nil {
		cfg.Level = zap.NewAtomicLevelAt(lvl)
	}
	return cfg.Build()
}

// IsDev: APP_ENV của dự án dùng "dev"/"prod"; chấp nhận thêm "development".
func IsDev(env string) bool {
	return strings.EqualFold(env, "dev") || strings.EqualFold(env, "development")
}

func WithTraceID(ctx context.Context, base *zap.Logger, traceID string) context.Context {
	ctx = context.WithValue(ctx, traceKey{}, traceID)
	return context.WithValue(ctx, loggerKey{}, base.With(zap.String("trace_id", traceID)))
}

func FromCtx(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
			return l
		}
	}
	return zap.L()
}

func TraceID(ctx context.Context) string {
	if ctx != nil {
		if s, ok := ctx.Value(traceKey{}).(string); ok {
			return s
		}
	}
	return ""
}

// Detach: ctx mới không bị huỷ theo request nhưng giữ logger/trace_id (cho goroutine nền).
func Detach(ctx context.Context) context.Context {
	base := context.Background()
	if id := TraceID(ctx); id != "" {
		base = context.WithValue(base, traceKey{}, id)
		base = context.WithValue(base, loggerKey{}, FromCtx(ctx))
	}
	return base
}
