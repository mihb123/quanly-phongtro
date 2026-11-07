package logging

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type stateKey struct{}

// RequestState là "hộp" dùng chung trong một request: tầng ngoài cùng (middleware log)
// tạo nó, các tầng bên trong (httpx.Fail, recover, auth) ghi vào, và middleware đọc lại
// sau khi handler chạy xong để log đúng MỘT dòng.
type RequestState struct {
	mu     sync.Mutex
	err    error
	stack  []byte
	userID string
	fields []zap.Field
	warns  []string
}

// WithRequestState gắn một RequestState mới vào ctx.
func WithRequestState(ctx context.Context) (context.Context, *RequestState) {
	st := &RequestState{}
	return context.WithValue(ctx, stateKey{}, st), st
}

func stateFrom(ctx context.Context) *RequestState {
	if ctx == nil {
		return nil
	}
	st, _ := ctx.Value(stateKey{}).(*RequestState)
	return st
}

// RecordError ghi lỗi của request (lỗi đầu tiên được giữ lại). Không có state thì bỏ qua.
func RecordError(ctx context.Context, err error) {
	if st := stateFrom(ctx); st != nil && err != nil {
		st.mu.Lock()
		if st.err == nil {
			st.err = err
		}
		st.mu.Unlock()
	}
}

// RecordPanic ghi lỗi và stack tại điểm panic.
func RecordPanic(ctx context.Context, err error, stack []byte) {
	if st := stateFrom(ctx); st != nil {
		st.mu.Lock()
		st.err, st.stack = err, stack
		st.mu.Unlock()
	}
}

// SetUserID ghi user đã xác thực để dòng log request có field user_id.
func SetUserID(ctx context.Context, userID string) {
	if st := stateFrom(ctx); st != nil {
		st.mu.Lock()
		st.userID = userID
		st.mu.Unlock()
	}
}

// Annotate gắn thêm field nghiệp vụ vào dòng log DUY NHẤT của request (vd. kết quả
// đối soát webhook). Đây không phải log: service/repo được phép gọi. Không có request
// state (code nền) thì field bị bỏ qua.
func Annotate(ctx context.Context, fields ...zap.Field) {
	if st := stateFrom(ctx); st != nil {
		st.mu.Lock()
		st.fields = append(st.fields, fields...)
		st.mu.Unlock()
	}
}

// RecordWarning ghi một lỗi KHÔNG làm hỏng request (best-effort: gửi tin Zalo, tải QR...).
// Trong request: gắn vào field `warnings` của dòng log duy nhất (mức tối thiểu WARN).
// Ngoài request (job/cron): log WARN ngay bằng logger trong ctx (ngoại lệ code nền).
func RecordWarning(ctx context.Context, msg string, err error) {
	if st := stateFrom(ctx); st != nil {
		text := msg
		if err != nil {
			text += ": " + err.Error()
		}
		st.mu.Lock()
		st.warns = append(st.warns, text)
		st.mu.Unlock()
		return
	}
	FromCtx(ctx).Warn(msg, ErrorFields(err)...)
}

// Snapshot là những gì các tầng bên trong đã ghi vào request state.
type Snapshot struct {
	Err        error
	PanicStack []byte
	UserID     string
	Fields     []zap.Field
	Warnings   []string
}

// Snapshot trả về bản sao những gì đã ghi.
func (s *RequestState) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{Err: s.err, PanicStack: s.stack, UserID: s.userID, Fields: append([]zap.Field(nil), s.fields...), Warnings: append([]string(nil), s.warns...)}
}
