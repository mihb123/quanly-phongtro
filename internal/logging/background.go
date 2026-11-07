package logging

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
	"go.uber.org/zap"
)

// Go chạy fn trong goroutine nền: ctx tách khỏi request (không bị huỷ theo request)
// nhưng giữ logger + trace_id, có timeout, recover panic, và tự log khi fn trả lỗi.
// Đây là ngoại lệ duy nhất của "log once": code nền không có request để trả lỗi lên.
func Go(parent context.Context, task string, timeout time.Duration, fn func(ctx context.Context) error) {
	bg := Detach(parent)
	bg = WithLogger(bg, FromCtx(bg).With(zap.String("task", task)))
	go func() {
		defer func() {
			if r := recover(); r != nil {
				FromCtx(bg).Error("background panic", zap.Any("panic", r), zap.Stack("stack"))
			}
		}()
		ctx, cancel := context.WithTimeout(bg, timeout)
		defer cancel()
		if err := fn(ctx); err != nil {
			LogTaskError(bg, "background task failed", err)
		}
	}()
}

// TaskContext tạo ctx cho một lần chạy tác vụ nền (cron, job) không xuất phát từ request:
// trace_id mới + logger gốc kèm field task.
func TaskContext(parent context.Context, task string) context.Context {
	return WithTraceID(parent, zap.L().With(zap.String("task", task)), uuid.NewString())
}

// WithLogger thay logger trong ctx (giữ nguyên trace_id).
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LogTaskError ghi lỗi của tác vụ nền (job, cron, goroutine) ở mức ERROR với đủ ngữ cảnh apperr.
func LogTaskError(ctx context.Context, msg string, err error, extra ...zap.Field) {
	FromCtx(ctx).Error(msg, append(extra, ErrorFields(err)...)...)
}

// ErrorFields: error + failed_layer/origin + stack (nếu có apperr).
func ErrorFields(err error) []zap.Field {
	if err == nil {
		return nil
	}
	fields := []zap.Field{zap.Error(err)}
	if layer := apperr.FailedLayer(err); layer != "" {
		fields = append(fields, zap.String("failed_layer", layer), zap.String("origin", apperr.Origin(err)))
	}
	if stack := apperr.Stack(err); len(stack) > 0 {
		fields = append(fields, zap.ByteString("stack", stack))
	}
	return fields
}
