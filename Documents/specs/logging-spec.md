# Đặc tả hệ thống Logging & Error handling ("Log once, wrap everywhere")

Tài liệu này dành cho agent/lập trình viên áp dụng hệ thống logging vào một dự án Go khác.
Implementation tham chiếu: framework **Fiber v2**, logger **zap**, kiến trúc **handler → service → repository**.
Module path ở dự án tham chiếu là `my-web-app`; khi áp dụng, thay bằng module path của dự án đích.

---

## 1. Mục tiêu và nguyên tắc

1. **Log once ở biên.** Chỉ tầng ngoài cùng (HTTP middleware) log lỗi của request. Repository/service **tuyệt đối không** gọi logger.
2. **Lỗi mang đủ ngữ cảnh tại nguồn.** Mỗi mắt xích lỗi biết tầng (`Layer`), hàm (`Op`), vị trí (`file:line`). Stack trace chụp **một lần duy nhất tại lỗi gốc**, không phải lúc log.
3. **Wrap có chủ đích**, luôn giữ `%w` để `errors.Is/As` bóc được lỗi gốc (sentinel, driver error).
4. **Client không bao giờ thấy lỗi nội bộ.** Lỗi không lường trước trả 500 với message chung; chi tiết chỉ nằm trong log.
5. **Mỗi request đúng một dòng log**, có `trace_id`, mức log theo status.
6. **Ngoại lệ duy nhất**: code chạy nền (goroutine, job, consumer) không có request để trả lỗi lên, được phép tự log, nhưng phải dùng chung logger và giữ `trace_id`.

## 2. Kiến trúc tổng quan

```
Request
  └─ LoggingMiddleware  (ngoài cùng: sinh trace_id, gọi ErrorHandler sớm, log 1 dòng)
       └─ RecoverMiddleware  (panic -> error + stack)
            └─ Handler  (return err, không tự trả err.Error())
                 └─ Service  (apperr.Service("Op", err))
                      └─ Repository  (apperr.Repo("Op", err))
```

Luồng lỗi: repo bọc lỗi driver → service bọc tiếp → handler `return err` → `ErrorHandler` map ra status/message an toàn → `LoggingMiddleware` log đủ chuỗi lỗi.

Ví dụ chuỗi lỗi trong log: `[service:ListPost] [repo:Query] failed to query posts: ... connection refused`.

## 3. Cấu trúc file cần tạo

| File | Vai trò |
|---|---|
| `pkg/apperr/apperr.go` | Error có Layer/Op/Caller/stack |
| `pkg/apperr/apperr_test.go` | Test apperr |
| `pkg/logger/logger.go` | Khởi tạo zap, truyền logger/trace_id qua context |
| `internal/handler/logging.go` | `LoggingMiddleware`, `RecoverMiddleware` |
| `internal/handler/errors.go` | `ErrorHandler`, bảng `errRules` |
| `internal/handler/logging_test.go` | Test middleware |
| `internal/lint/wrap_test.go` | Test AST chặn lỗi chưa wrap |
| `internal/config/config.go` | Thêm `AppEnv`, `LogLevel` |
| `cmd/api/main.go` | Nối dây |

Dependency: `go get go.uber.org/zap@v1.27.0`; `github.com/google/uuid` (sinh trace_id).

## 4. Code tham chiếu

### 4.1 `pkg/apperr/apperr.go`

```go
// Package apperr cung cấp error mang thông tin tầng (Layer), tên hàm (Op) và vị trí
// file:line. Stack trace chỉ được chụp MỘT LẦN, tại mắt xích đầu tiên (lỗi gốc) của chuỗi.
package apperr

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

const (
	LayerRepo    = "repo"
	LayerService = "service"
	LayerHandler = "handler"
)

type Error struct {
	Layer  string
	Op     string
	Err    error
	Caller string // file:line nơi mắt xích này được tạo
	stack  []byte // chỉ khác nil ở lỗi gốc
}

func (e *Error) Error() string { return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Op, e.Err) }
func (e *Error) Unwrap() error { return e.Err }

// Wrap bọc err. Trả về nil nếu err == nil.
func Wrap(layer, op string, err error) error { return wrap(3, layer, op, err) }

// Wrapf như Wrap, thêm ngữ cảnh: "[layer:op] ctx: err".
func Wrapf(layer, op string, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return wrap(3, layer, op, fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
}

// New tạo lỗi gốc (không có cause).
func New(layer, op, msg string) error { return wrap(3, layer, op, errors.New(msg)) }

func Repo(op string, err error) error    { return wrap(3, LayerRepo, op, err) }
func Service(op string, err error) error { return wrap(3, LayerService, op, err) }

func wrap(skip int, layer, op string, err error) error {
	if err == nil {
		return nil
	}
	e := &Error{Layer: layer, Op: op, Err: err, Caller: caller(skip)}
	var inner *Error
	if !errors.As(err, &inner) { // lỗi gốc: chụp stack đúng một lần
		e.stack = debug.Stack()
	}
	return e
}

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/%s:%d", filepath.Base(filepath.Dir(file)), filepath.Base(file), line)
}

func root(err error) *Error {
	var last *Error
	for err != nil {
		var ae *Error
		if !errors.As(err, &ae) {
			break
		}
		last = ae
		err = ae.Err
	}
	return last
}

// FailedLayer: tầng sâu nhất trong chuỗi ("" nếu không có apperr).
func FailedLayer(err error) string {
	if r := root(err); r != nil {
		return r.Layer
	}
	return ""
}

// Origin: file:line của mắt xích sâu nhất.
func Origin(err error) string {
	if r := root(err); r != nil {
		return r.Caller
	}
	return ""
}

// Stack: stack trace chụp tại lỗi gốc (nil nếu không có).
func Stack(err error) []byte {
	if r := root(err); r != nil {
		return r.stack
	}
	return nil
}
```

> Quan trọng: `skip = 3` tính từ `caller()` (0=caller, 1=wrap, 2=Repo/Wrap/..., 3=nơi gọi). Mọi hàm public gọi `wrap` **trực tiếp** để giữ đúng độ sâu; đừng thêm lớp trung gian.

### 4.2 `pkg/logger/logger.go`

```go
package logger

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}
type traceKey struct{}

// New: JSON ở production, console ở development.
func New(env, level string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.EqualFold(env, "production") {
		cfg = zap.NewProductionConfig()
	} else {
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
```

### 4.3 `internal/handler/logging.go` (Fiber)

```go
package handler

import (
	"runtime/debug"
	"time"

	"<module>/pkg/apperr"
	"<module>/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	TraceIDHeader = "X-Trace-ID"
	localsStack   = "_panic_stack"
)

func LoggingMiddleware(base *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		traceID := c.Get(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.NewString()
		}
		c.Set(TraceIDHeader, traceID)
		c.SetUserContext(logger.WithTraceID(c.UserContext(), base, traceID))

		chainErr := c.Next()
		if chainErr != nil {
			// Fiber chỉ gọi ErrorHandler sau khi chain kết thúc; gọi sớm để biết status cuối cùng.
			if hErr := c.App().ErrorHandler(c, chainErr); hErr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		status := c.Response().StatusCode()
		fields := []zap.Field{
			zap.String("trace_id", traceID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Int("bytes", len(c.Response().Body())),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.IP()),
		}
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			fields = append(fields, zap.String("user_id", uid))
		}

		err := chainErr
		if err == nil {
			err, _ = c.Locals(localsErrKey).(error)
		}
		var stack []byte
		if err != nil {
			fields = append(fields, zap.String("error", err.Error()))
			if layer := apperr.FailedLayer(err); layer != "" {
				fields = append(fields, zap.String("failed_layer", layer), zap.String("origin", apperr.Origin(err)))
			}
			stack = apperr.Stack(err)
			if stack == nil {
				stack, _ = c.Locals(localsStack).([]byte) // panic
			}
		}

		switch {
		case status >= 500:
			if len(stack) > 0 {
				fields = append(fields, zap.ByteString("stack", stack))
			}
			base.Error("request failed", fields...)
		case status >= 400:
			base.Warn("request rejected", fields...)
		default:
			base.Info("request completed", fields...)
		}
		return nil
	}
}

func RecoverMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, _ interface{}) {
			c.Locals(localsStack, debug.Stack())
		},
	})
}
```

### 4.4 `internal/handler/errors.go`

```go
package handler

import (
	"errors"

	"<module>/pkg/response" // hoặc helper JSON error của dự án
	// import package chứa sentinel error của dự án

	"github.com/gofiber/fiber/v2"
)

const localsErrKey = "_err"

type errRule struct {
	target  error
	status  int
	message string
}

// errRules: sentinel error -> HTTP status + message an toàn cho client.
// ĐIỀU CHỈNH theo sentinel của dự án đích. Thêm sentinel mới = thêm 1 dòng ở đây.
var errRules = []errRule{
	// {service.ErrInvalidInput, fiber.StatusBadRequest, "Invalid input data"},
	// {repository.ErrNotFound, fiber.StatusNotFound, "Resource not found"},
	// {service.ErrEmailExists, fiber.StatusConflict, "Email already exists"},
}

// ErrorHandler: chỉ map lỗi -> response, KHÔNG log.
func ErrorHandler(c *fiber.Ctx, err error) error {
	c.Locals(localsErrKey, err)

	var fe *fiber.Error
	if errors.As(err, &fe) {
		if fe.Code >= fiber.StatusInternalServerError {
			return response.JSONError(c, fe.Code, "Internal server error")
		}
		return response.JSONError(c, fe.Code, fe.Message)
	}
	for _, r := range errRules {
		if errors.Is(err, r.target) {
			return response.JSONError(c, r.status, r.message)
		}
	}
	return response.JSONError(c, fiber.StatusInternalServerError, "Internal server error")
}
```

### 4.5 `internal/lint/wrap_test.go`

```go
package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// TestErrorsAreWrapped: repository/service không được trả lỗi "trần".
// Trần = return <ident err> | return <ErrXxx sentinel> | return fmt.Errorf(...) / errors.New(...).
func TestErrorsAreWrapped(t *testing.T) {
	for _, dir := range []string{"../repository", "../service"} { // chỉnh theo cấu trúc dự án đích
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				ast.Inspect(f, func(n ast.Node) bool {
					ret, ok := n.(*ast.ReturnStmt)
					if !ok || len(ret.Results) == 0 {
						return true
					}
					if reason := bare(ret.Results[len(ret.Results)-1]); reason != "" {
						t.Errorf("%s: %s", fset.Position(ret.Pos()), reason)
					}
					return true
				})
			}
		}
	}
}

func bare(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		if x.Name == "err" {
			return "bare `err` must be wrapped with apperr.Repo/Service/Wrap"
		}
		if strings.HasPrefix(x.Name, "Err") {
			return "sentinel " + x.Name + " must be wrapped with apperr"
		}
	case *ast.SelectorExpr:
		if strings.HasPrefix(x.Sel.Name, "Err") {
			return "sentinel " + x.Sel.Name + " must be wrapped with apperr"
		}
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				if (id.Name == "fmt" && sel.Sel.Name == "Errorf") || (id.Name == "errors" && sel.Sel.Name == "New") {
					return id.Name + "." + sel.Sel.Name + " must be wrapped with apperr"
				}
			}
		}
	}
	return ""
}
```

### 4.6 Nối dây

`internal/config/config.go`: thêm `AppEnv` (`APP_ENV`, mặc định `development`) và `LogLevel` (`LOG_LEVEL`, mặc định `info`).

`cmd/api/main.go`:

```go
zl, err := logger.New(cfg.AppEnv, cfg.LogLevel)
if err != nil { panic("failed to init logger: " + err.Error()) }
defer zl.Sync()
zap.ReplaceGlobals(zl)

app := fiber.New(fiber.Config{ErrorHandler: handler.ErrorHandler})
app.Use(handler.LoggingMiddleware(zl)) // ngoài cùng
app.Use(handler.RecoverMiddleware())
// cors, auth, routes...
```

Thay mọi `log.Printf/Fatalf` ở `main.go` bằng `zl.Info/Warn/Fatal(..., zap.Error(err))`. Bỏ `fiber/middleware/logger` mặc định.

## 5. Quy ước viết code (BẮT BUỘC)

### Repository
```go
func (r *userRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Repo("GetByID", ErrNotFound)      // sentinel PHẢI được wrap
	}
	if err != nil {
		return nil, apperr.Repo("GetByID", fmt.Errorf("id=%d: %w", id, err)) // thêm ngữ cảnh, giữ %w
	}
	return &u, nil
}
```

### Service
```go
func (s *postService) Get(ctx context.Context, id int64) (*Post, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Service("Get", err)
	}
	if p.Title == "" {
		return nil, apperr.Service("Get", ErrInvalidInput)
	}
	return p, nil
}
```

### Handler
```go
res, err := h.postService.Get(c.UserContext(), id)   // KHÔNG dùng c.Context()
if err != nil {
	return err        // ErrorHandler map status; middleware log. KHÔNG trả err.Error() cho client.
}
```
Nhánh 4xx cần message riêng theo ngữ cảnh vẫn có thể `errors.Is` rồi trả JSON trực tiếp; chỉ nhánh 500/không lường trước mới `return err`.

### Code chạy nền
```go
func (s *svc) async(parent context.Context, name string, fn func(ctx context.Context) error) {
	bg := logger.Detach(parent)               // giữ logger + trace_id, không bị huỷ theo request
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.FromCtx(bg).Error("background panic", zap.String("task", name), zap.Any("panic", r), zap.Stack("stack"))
			}
		}()
		ctx, cancel := context.WithTimeout(bg, 10*time.Second)
		defer cancel()
		if err := fn(ctx); err != nil {
			logger.FromCtx(bg).Error("background task failed", zap.String("task", name),
				zap.String("failed_layer", apperr.FailedLayer(err)), zap.String("origin", apperr.Origin(err)),
				zap.ByteString("stack", apperr.Stack(err)), zap.Error(err))
		}
	}()
}
```
Mọi hàm chạy nền nhận `ctx context.Context` từ request để truyền vào `Detach`.

### Cấm
- `log.Printf`, `fmt.Println`, `logger.Error(...)` trong repo/service.
- `return err` trần, `return ErrXxx` trần, `return fmt.Errorf(...)` trần trong repo/service (lint sẽ fail).
- Trả `err.Error()` của lỗi nội bộ cho client.
- Log body request, header `Authorization`, mật khẩu, token, PII.
- Dùng `fmt.Errorf` với `%v` thay `%w`.

## 6. Spec output log

Một dòng log mỗi request:

| Field | Ghi chú |
|---|---|
| `timestamp`, `level` | zap, ISO8601 |
| `trace_id` | từ `X-Trace-ID` hoặc uuid mới; trả lại ở response header |
| `method`, `path`, `status`, `bytes`, `latency`, `ip` | luôn có |
| `user_id` | nếu đã xác thực |
| `error` | cả chuỗi `[service:..] [repo:..] ...` (khi có lỗi) |
| `failed_layer`, `origin` | tầng và `file:line` của lỗi gốc (khi có apperr) |
| `stack` | chỉ 5xx; stack tại lỗi gốc, hoặc tại điểm panic |

Mức log: 5xx → ERROR, 4xx → WARN, còn lại → INFO. Message: `request failed` / `request rejected` / `request completed`.

## 7. Quy trình áp dụng cho dự án đích (checklist cho agent)

1. Đọc cấu trúc dự án: framework HTTP, các tầng, chỗ đang dùng `log.*`, danh sách sentinel error, helper response JSON. **Nếu framework không phải Fiber**, giữ nguyên `apperr` và `logger`, viết lại middleware + error handler theo framework đó (giữ cùng hành vi ở mục 6).
2. Thêm dependency zap và uuid.
3. Tạo `pkg/apperr`, `pkg/logger` (mục 4.1, 4.2) và chạy test của apperr.
4. Tạo middleware và `ErrorHandler` (4.3, 4.4), điền `errRules` bằng sentinel thật của dự án.
5. Thêm config `APP_ENV`, `LOG_LEVEL`; nối dây trong `main.go` (4.6).
6. Chạy test lint (4.5) để liệt kê mọi `return` chưa wrap; **sửa từng chỗ** theo quy ước mục 5 (tên `Op` = tên hàm chứa nó). Lặp lại tới khi lint xanh.
7. Handler: đổi các nhánh 500 trả `err.Error()` thành `return err`; đổi `c.Context()` thành `c.UserContext()` khi gọi xuống service (trừ chỗ cần fasthttp context như SSE stream writer).
8. Code chạy nền: truyền `ctx` vào, dùng `logger.Detach`; đổi `log.*` sang zap.
9. Xoá mọi `log.*` stdlib còn lại ở repo/service.
10. Thêm test middleware (mục 8) và mục "Logging & error convention" vào README.
11. Chạy `go build ./... && go vet ./... && go test ./...`.

## 8. Tiêu chí nghiệm thu

Test middleware (dùng `zaptest/observer` + `app.Test()`) phải kiểm tra:

- `/ok` → 200, đúng 1 log INFO.
- Lỗi sentinel 404 → đúng 1 log WARN.
- Lỗi DB (5xx) → đúng 1 log ERROR; `failed_layer == "repo"`; `error` chứa `[service:...] [repo:...]`; `origin` trỏ về nơi tạo lỗi gốc; `stack` chứa tên hàm tạo lỗi gốc (không phải middleware).
- Panic → 500, 1 log ERROR, có `stack` tại điểm panic.
- Response 5xx **không chứa** chuỗi lỗi nội bộ (vd. "connection refused").
- `X-Trace-ID` được sinh khi thiếu và được tái sử dụng khi client gửi lên.
- `go test ./internal/lint` xanh (không còn lỗi trần ở repo/service).

## 9. Điểm cần điều chỉnh theo dự án đích

- Module path trong import (`<module>`).
- Bảng `errRules` (sentinel → status/message).
- Helper response JSON (`response.JSONError`) hoặc dùng `c.Status(...).JSON(...)`.
- Thư mục cần lint (`../repository`, `../service`) và tên tầng nếu khác (thêm hằng `Layer*` tương ứng).
- Cách lấy `user_id` (ở đây là `c.Locals("user_id")` kiểu string).
- Nếu có SSE/WebSocket/stream dài: middleware chỉ log lúc mở kết nối; không log từng event.

## 10. Hạn chế đã biết

- `failed_layer` chỉ chính xác với lỗi đi qua `apperr`; lỗi sinh ở handler (`BodyParser`, parse id) không có layer.
- Nhánh 4xx handler tự trả JSON (không `return err`) sẽ log WARN nhưng không có field `error`.
- Stack chụp bằng `debug.Stack()` cho mỗi lỗi gốc, có chi phí; chấp nhận được vì chỉ phát sinh khi có lỗi.
- Audit log (hành động nghiệp vụ như đăng nhập, thanh toán) **chưa nằm trong phạm vi** tài liệu này.
