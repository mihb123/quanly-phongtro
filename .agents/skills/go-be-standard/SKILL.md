---
name: go-be-standard
description: Chuẩn BẮT BUỘC khi viết/sửa Go backend của quanly-phongtro (chi + net/http, bun/PostgreSQL) — input decode + validate xong ở request struct của handler (httpx.Bind, tag json/query/path + validate + label tiếng Việt) rồi mới vào Service; Service chứa nghiệp vụ, trả (T, error) với lỗi nghiệp vụ apperr tiếng Việt, lỗi hệ thống wrap %w; Handler mỏng, mọi lỗi qua httpx.Fail (HTTP status thật, log đúng 1 lần, không lộ lỗi hệ thống); Repository chỉ query, luôn giới hạn theo manager, placeholder, ngày [from, nextDay), chạy được trong transaction. Dùng khi thêm/sửa endpoint, handler, service, repository, request/response struct, bộ lọc, phân trang, query, transaction — kể cả khi người dùng không nhắc.
---

# Go backend standard (quanly-phongtro)

```
HTTP ─► middleware ─► Handler ─────────────► Service ─────────────► Repository ─► PostgreSQL
        JWT/DPoP      GetManagerID, Bind      nghiệp vụ, quyền theo   chỉ query, lọc manager,
        role/admin    req.input()/filter()    bản ghi, transaction    database.Executor
                      httpx.Fail / WriteJSON  (T, error) + apperr     ErrNoRows → sentinel
```

**Validate xong rồi mới vào Service.** Service nhận struct input đã đúng kiểu (`time.Time`, `int64`, chuỗi đã trim) và tin dữ liệu đó. Không truyền `*http.Request`, `url.Values`, `map[string]any` hay body thô xuống dưới.

## 0. Trước khi viết

1. `AGENTS.md` (= `CLAUDE.md`) của repo thắng skill khi mâu thuẫn. Đọc phần *Gotchas* trong đó.
2. Dùng cái đã có, không tạo bản song song:
   - Response: `httpx.WriteJSON(w, status, data, message)` / `httpx.WriteError(w, status, message)`, envelope `{status:int, data, message}` (frontend đọc `error.response.data.message`). **Không** đổi sang envelope khác (`status:bool`, `error_code`…).
   - Lấy user: `httpx.GetManagerID(r, w)` — nó **đã ghi 401**, chỉ `return` (đừng `WriteError` thêm lần nữa như vài handler cũ).
   - Transaction: `database.WithinTransaction(ctx, db, fn)` + `database.Executor(ctx, r.db)` (`internal/db`).
   - Logger: `internal/service/logger` (`logger.Error(r, status, msg, err)`).
   - Rule validate riêng: `housecode`, `monthperiod` đã có.
   - bun **v1.2.18**: `Limit(int)`, `Offset(int)`, `ScanAndCount` trả `int`.
3. Chưa có `httpx.Bind` / `httpx.Fail` / `internal/apperr` → copy `references/kit/` một lần (`apperr/` → `internal/apperr/`, `httpx/*.go` → `internal/handler/httpx/`), đã khớp module + envelope + logger của repo, không đụng `ValidateStruct` cũ.
4. Chỉ chuẩn hoá endpoint đang làm. Handler cũ (decode tay, message tiếng Anh) giữ nguyên nếu không được yêu cầu refactor.
5. Không thêm framework/thư viện (gin, echo, gorm, viper, ozzo…). Validator v10, chi, bun, lib/pq đã có. Module `pkg/<name>` là stdlib thuần: không import kit/validator, request tự viết `Validate()` trả error.

Mẫu đầy đủ một domain (model → repository → service → handler + DTO): `references/example/`.

## 1. Handler — nhận request, trả response

- Mỗi endpoint một request struct unexported (`<domain>_dto.go` cạnh handler) + response struct riêng. Không decode thẳng vào model hay input của service (client tự gán `manager_id`, `status`…).
- Tag: `json` (body), `query`, `path` (đọc `r.PathValue`, chi 5.2 đã set), `label` (tên tiếng Việt trong câu lỗi), `validate`. Field query/path đặt `json:"-"` để body không ghi đè; `bind:"raw"` để không trim (mật khẩu).
- `httpx.Bind(w, r, &req)`: decode JSON (≤ 1MB, chặn dữ liệu thừa sau object), query (`ids=1&ids=2` và `ids[]=` cho slice; field đơn nhận nhiều giá trị → lỗi), path (gán cuối, body không giả được id), trim chuỗi, validate tag, rồi `Validate()`. Trả `false` = đã ghi response: 422 `{status, message: lỗi đầu tiên, errors: {"items.0.id": "…"}}`, JSON hỏng 400, quá lớn 413. Chuỗi rỗng/khoảng trắng trong query = không gửi.
- Rule hay dùng: `required`, `omitempty`, `uuid` (**mọi** id uuid — chuỗi rỗng/sai vào cột uuid là 500), `min/max/len`, `gt/gte/lt/lte`, `oneof=25 50 100`, `datetime=2006-01-02`, `monthperiod`, `email`, `e164`, `unique`, `dive`. Rule mới → `httpx.RegisterRule("tag", ":field …", fn)` trong `init()` của httpx; không validate tay rải rác ở service.
- Rule chéo field / parse phức tạp → method `Validate(errs *httpx.FieldErrors)` (chỉ chạy khi tag đã hợp lệ), lưu kết quả parse vào field unexported. Request → `req.input()` / `req.filter()` trả struct đúng kiểu, có default (page 1, limit 25).
- Ngày lọc: query `YYYY-MM-DD` + `datetime=2006-01-02`; `Validate` kiểm `to >= from`; `httpx.DayRange(from, to, tz)` → 00:00 ngày đầu và 00:00 **ngày kế tiếp** theo giờ nghiệp vụ `time.FixedZone("ICT", 7*3600)` (không `LoadLocation`: binary có thể thiếu tzdata). Thời điểm chính xác: RFC3339, parse trong `Validate`, `.UTC()`.
- Handler chỉ: `GetManagerID` → `Bind` → gọi service với `r.Context()` → lỗi `httpx.Fail(w, r, err)` → thành công map response DTO → `WriteJSON`. Không query, không nghiệp vụ, không mở transaction, không tin `manager_id/user_id/role` từ client, không ghi dữ liệu qua `GET`, không `WriteError(..., err.Error())`.
- Response: không trả model bun (lộ cột nội bộ, đổi schema là vỡ API). List `make([]T, 0, n)` để ra `[]` chứ không `null`. Tiền/số lượng không `omitempty`. Bảng/field tiền **mới** dùng `int64` VND; field cũ đang `float64` (invoice, house cost…) thì theo field đó, không trộn trong cùng domain.
- Danh sách phân trang ở backend: `limit` ∈ 25/50/100, trả `items, total, page, limit` (+ `totals` của toàn tập lọc nếu UI có dòng Tổng).

```go
func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}
	var req createExpenseRequest
	if !httpx.Bind(w, r, &req) {
		return
	}
	expense, err := h.service.Create(r.Context(), managerID, req.input())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toExpenseResponse(expense), "Đã thêm khoản chi")
}
```

## 2. Phân quyền

- Đăng nhập/token: middleware (JWT + DPoP). Thiếu/sai → **401**. Luôn dùng HTTP status thật, không trả 200 kèm lỗi: axios chỉ reject non-2xx, 200 + lỗi bị frontend coi là thành công.
- Quyền theo vai trò (vd `PLUGIN_ADMIN_EMAILS`): middleware hoặc đầu handler → 403.
- Quyền theo bản ghi (nhà/phòng/hoá đơn của manager khác): cần DB → repository luôn lọc theo `managerID` **và/hoặc** service kiểm (`IsHouseOwnedBy`). Không thuộc manager → `NotFound` (không lộ bản ghi tồn tại) hoặc `Forbidden` theo domain.

## 3. Service — toàn bộ nghiệp vụ

- Tính toán, chuyển trạng thái, kiểm tra cần DB (tồn tại, trùng, thuộc manager), transaction, gọi API ngoài (đặt `context.WithTimeout`).
- Chữ ký `Create(ctx context.Context, managerID string, in CreateInput) (*model.Expense, error)`: `ctx` đầu tiên, không lưu ctx trong struct, không import `net/http`, không trả status code, không trim/parse lại input.
- Theo quy ước repo: service là interface + `xxxServiceImpl`, phụ thuộc interface repository trong `model` (mock bằng mockgen, `make mocks`). Không thêm interface mới ngoài quy ước đó.

| Tình huống | Service trả | `httpx.Fail` ra |
|---|---|---|
| Thành công | `value, nil` | handler `WriteJSON` 200/201 |
| Không tìm thấy | `model.ErrXxxNotFound` = `apperr.NotFound("Không tìm thấy …")` | 404 + message |
| Vi phạm nghiệp vụ | `apperr.Invalid("Hoá đơn đã thanh toán, không sửa được")` | 422 + message |
| Không có quyền với bản ghi | `apperr.Forbidden(…)` / `NotFound` | 403 / 404 |
| Trùng (unique, sai trạng thái) | `apperr.Conflict(…)` | 409 |
| Lỗi hệ thống (DB, API ngoài) | `fmt.Errorf("ngữ cảnh: %w", err)` | log 1 lần + 500 "Có lỗi xảy ra, vui lòng thử lại" |

- Lỗi nghiệp vụ khai `var ErrXxx = apperr.X("…")` trong `model` (như sentinel cũ của repo), so bằng `errors.Is` (bọc `%w` hay `.Wrap(cause)` vẫn khớp). Message tiếng Việt, cụ thể, người dùng tự sửa được. `apperr` không chứa HTTP status: map status chỉ ở `httpx.Fail`.
- **Log đúng một lần** ở `httpx.Fail`. Service/repository không log rồi trả lỗi; chỉ wrap khi thêm ngữ cảnh có ích. Ngoại lệ: job nền/cron không có handler thì tự log.
- Không `panic` cho nghiệp vụ, không so sánh chuỗi lỗi.
- Transaction: nhiều bước ghi phải nguyên tử → service (hoặc repo nếu gói gọn trong một hàm repo, như `invoice_repository`) bọc `database.WithinTransaction(ctx, db, fn)`; mọi method repo dùng `database.Executor(ctx, r.db)` nên tự chạy trong tx. Không truyền `bun.Tx` qua tham số. Không gọi API ngoài bên trong tx.
- Goroutine nền sinh từ request: `context.WithoutCancel(ctx)` + `WithTimeout`, `recover`, log lỗi; không dùng `r.Context()`/`ResponseWriter` sau khi handler trả về. Không spawn goroutine không kiểm soát vòng đời.
- Hằng nghiệp vụ cấu hình được (hạn mức, phí, TTL) lấy từ config inject vào, không hardcode.

## 4. Repository — chỉ query DB

- Mọi method: `ctx` đầu tiên, query qua `database.Executor(ctx, r.db)`, nhận filter đã đúng kiểu (`if !f.From.IsZero() { q = q.Where("spent_at >= ?", f.From) }`), ngày `>= from AND < to` (timestamptz có micro giây, không `23:59:59`).
- Mọi query dữ liệu của manager giới hạn theo `managerID` (cột trực tiếp hoặc `house_id IN (SELECT id FROM houses WHERE manager_id = ?)`), kể cả update/delete.
- Placeholder `?`; không `fmt.Sprintf` giá trị vào SQL; cột sort động → map whitelist.
- `sql.ErrNoRows` → sentinel `model.ErrXxxNotFound`; unique → `errors.As(err, &pqErr) && pqErr.Code == "23505"` → sentinel Conflict (không `strings.Contains(err.Error(), …)`); update/delete kiểm `RowsAffected()` khi cần biết có bản ghi; lỗi khác `fmt.Errorf("select expense %s: %w", id, err)`.
- Không trim/parse/validate (`strings.TrimSpace`, `strconv`, `time.Parse`), không nghiệp vụ, không đọc auth từ context. Thấy cần → request đang thiếu rule, sửa ở đó.
- Chống N+1: lấy theo lô `WHERE id IN (?)` + `bun.In`, hoặc join/relation; không query trong vòng lặp. Danh sách luôn có `LIMIT`; đếm bằng `ScanAndCount`.
- bun + cột có default DB (`uuidv7()`, `now()`): `ExcludeColumn("id", …)` + `Returning("id, …")`.
- SQL thô qua `database/sql`: `defer rows.Close()`, kiểm `rows.Err()` sau vòng `rows.Next()`.

## 5. Cấu trúc Go

- Package theo domain (`invoice`, `house`), không `utils/common/helpers`. Integration bên thứ ba → `pkg/<name>` + một file `<name>_adapter.go` (xem AGENTS.md).
- **Không tạo package/file/type/interface cho thứ chỉ 1 nơi dùng**: grep trước; chỉ 1 nơi → hàm unexported trong file dùng nó. Tách tầng handler → service → repository luôn giữ.
- Biến thể mới (ngân hàng, cổng thanh toán, loại báo cáo) → implement interface/func + đăng ký vào registry (`map[code]Parser`, plugin), không nối dài `switch`.
- DI qua constructor `NewX(deps…)` nối ở `cmd/api/main.go` / router; không biến global mutable, `init()` chỉ đăng ký rule/plugin.
- Hàm làm một việc, early return, không tham số `bool` đổi hành vi. Map/cache phải có giới hạn hoặc cơ chế dọn. `sync.Mutex`/`WaitGroup` không copy, dùng con trỏ.

## 6. Kiểm tra

- `gofmt -l <files>`, `go vet ./<pkg>/...`, `go build ./...`; `staticcheck ./<pkg>/...` (đã cài).
- `.agents/skills/go-be-standard/scan-go-be.sh`: không tham số = chỉ soát **dòng thêm/sửa** so với HEAD (+ file chưa track); truyền file/thư mục để soát toàn bộ. Mỗi dòng là gợi ý: sửa, hoặc nói rõ vì sao giữ.
- Package đã sửa có test sẵn → chạy `go test ./<pkg>/...` của đúng package đó (sqlmock assert SQL chính xác, đổi query là test cũ vỡ → cập nhật regex `^…$`, không nới thành `.*`). Không tự viết test mới, không chạy full suite/Playwright trừ khi user yêu cầu.

## Checklist

- [ ] Dùng helper sẵn có (`WriteJSON/WriteError`, `GetManagerID`, `Bind/Fail`, `WithinTransaction/Executor`, logger); không thêm bản song song, không thêm thư viện.
- [ ] Mọi input qua request struct + `httpx.Bind` trước khi vào service; field có `label` tiếng Việt; id uuid có `uuid`; ngày lọc `datetime` + `DayRange`; rule chéo field ở `Validate()`.
- [ ] Input xấu (`2026-02-30`, `?page[]=1`, `page=x`, khoảng ngày ngược, JSON hỏng/thừa, sai kiểu, id không phải uuid) → 4xx message tiếng Việt, không 500.
- [ ] Service trả `(T, error)`; lỗi nghiệp vụ `apperr` tiếng Việt trong `model`; lỗi hệ thống wrap `%w`; chỉ `Fail` log; người dùng không thấy chi tiết DB/stack.
- [ ] Handler không query/nghiệp vụ/transaction; manager lấy từ JWT, không từ body; không ghi response 2 lần; response qua DTO, list `[]`, tiền không `omitempty`.
- [ ] Repository lọc theo manager, placeholder, `< nextDay`, `Executor`, `ErrNoRows`/`23505` → sentinel, không N+1, có `LIMIT`.
- [ ] Không có package/type/interface mới chỉ 1 nơi dùng.
- [ ] `gofmt`, `go vet`, `go build` sạch; `scan-go-be.sh` đã xem hết.
