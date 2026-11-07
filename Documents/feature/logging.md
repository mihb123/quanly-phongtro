# Logging & error convention ("log once, wrap everywhere")

Spec gốc: `Documents/specs/logging-spec.md` (viết cho Fiber). Dự án dùng chi nên phần middleware/error handler được viết lại, hành vi output giữ nguyên.

## Thành phần

| Ở đâu | Vai trò |
|---|---|
| `pkg/apperr` | Error có `Layer`/`Op`/`Caller`, stack chụp **một lần** ở lỗi gốc. `apperr.Repo/Service/Wrap/Wrapf/New`, `FailedLayer`, `Origin`, `Stack`, `Message` (bỏ tiền tố `[layer:op]`). Chỉ dùng stdlib. |
| `internal/logging` | zap (`New(APP_ENV, LOG_LEVEL)`: `dev` → console, còn lại → JSON), `WithTraceID/FromCtx/TraceID/Detach`, request state (`RecordError`, `RecordWarning`, `Annotate`, `SetUserID`), tác vụ nền (`Go`, `TaskContext`, `LogTaskError`). Nằm ở `internal/` vì phụ thuộc zap. |
| `internal/logging/filelog` | Log file JSONL chuyên biệt: slow API (`SLOW_API_*`) và security (abuse monitor). Không đổi format. |
| `internal/router/logging.go` | `loggingMiddleware` (ngoài cùng) + `recoverMiddleware` (ngay bên trong). |
| `internal/handler/httpx/errors.go` | `errRules` (sentinel → status), `Fail`, `FailWith`, `NoteError`, `StatusOf`, `PublicMessage`. |
| `internal/lint/wrap_test.go` | Test AST: repository/service không được `return err` / `return ErrX` / `return fmt.Errorf(...)` / `return errors.New(...)` trần. |

Config: `APP_ENV` (đã có, `dev`/`prod`, mặc định `prod`) và `LOG_LEVEL` (mặc định `info`).

## Một dòng log mỗi request

`timestamp`, `level`, `msg`, `trace_id` (từ header `X-Trace-ID` nếu hợp lệ `[A-Za-z0-9._-]{1,64}`, không thì uuid mới; luôn trả lại ở response header), `method`, `path`, `route` (pattern chi), `status`, `bytes`, `latency` (giây), `ip` (client IP thật qua trusted proxy), `user_id` (khi đã xác thực), `error`, `failed_layer` + `origin` (khi có apperr), `stack` (chỉ 5xx: stack tại lỗi gốc hoặc tại điểm panic), `warnings`, và các field nghiệp vụ do `Annotate` thêm.

Mức log: 5xx → ERROR `request failed`; 4xx → WARN `request rejected`; 2xx/3xx có lỗi/warning được ghi nhận → WARN `request completed with warnings`; còn lại → INFO `request completed`.

## Quy ước theo tầng

**Repository / service**: không gọi logger. Mọi lỗi trả về đều wrap, `Op` = tên hàm chứa nó:

```go
if errors.Is(err, sql.ErrNoRows) {
	return nil, apperr.Repo("GetByID", model.ErrHouseNotFound)
}
if err != nil {
	return nil, apperr.Repo("GetByID", fmt.Errorf("get house %s: %w", id, err)) // luôn %w, không %v
}
```

- So sánh lỗi bằng `errors.Is/As`, **không** dùng `==` (lỗi đã bị wrap).
- Lỗi đầu vào có câu chữ cho người dùng: `sharedsvc.Invalid("...")` hoặc `sharedsvc.InvalidCause("...", cause)` (400, client chỉ thấy câu chữ, log thấy cả cause).
- Lỗi không làm hỏng request (gửi Zalo thất bại, tải QR lỗi...): `logging.RecordWarning(ctx, "msg", err)`. Trong request nó vào field `warnings`; ngoài request (job/cron) nó log WARN ngay.
- Kết quả nghiệp vụ muốn thấy trong log (vd. `payment_outcome=underpaid`): `logging.Annotate(ctx, zap.String(...))`. Không đưa PII/nội dung tin nhắn vào.

**Handler**:

```go
res, err := h.service.Get(r.Context(), id)
if err != nil {
	httpx.Fail(w, r, err) // errRules → status + message an toàn; không khớp → 500 "internal server error"
	return
}
```

- Nhánh 4xx cần câu chữ riêng: `httpx.FailWith(w, r, status, "message", err)`.
- Webhook phải trả 200 cho nhà cung cấp: `httpx.NoteError(r, err)` rồi ghi 200.
- Không bao giờ trả `err.Error()` của lỗi từ service. Câu chữ lỗi validate do chính handler sinh ra (validator, parse ngày) thì được trả.
- Thêm sentinel mới = thêm 1 dòng vào `errRules`.

**Code chạy nền**: `logging.Go(ctx, "task", timeout, fn)` (tách ctx khỏi request nhưng giữ `trace_id`, có recover và tự log lỗi). Cron/job tạo ctx bằng `logging.TaskContext(ctx, "task")` và log lỗi bằng `logging.LogTaskError`. `jobs.Worker` tự recover panic của từng job.

**`pkg/<integration>`**: không log. Báo cho app qua giá trị trả về hoặc hook (vd. `sepay.Reconciler.OnSkip`, `sepay.IsUnauthenticated`). App nối hook trong `<name>_adapter.go`.

**Plugin**: dùng `plugin.Warn(ctx, msg, err)`, không import logger.

## Cấm

- `log.Printf`, `fmt.Println`, logger trong repository/service (trừ `RecordWarning`/`Annotate` vốn chỉ ghi vào dòng log của request).
- `return err` trần, `return ErrX` trần, `return fmt.Errorf(...)` trần trong repo/service (`go test ./internal/lint` fail).
- Log body request, header `Authorization`, mật khẩu, token, email, nội dung tin nhắn.
