---
name: go-be-standard
description: Chuẩn bắt buộc khi viết/sửa code Go backend (Handler/DTO, UseCase/Service, Repository, query DB, xử lý lỗi, response chuẩn). Dùng khi thêm/sửa endpoint, DTO, UseCase, Repository hoặc query lọc theo ngày — kể cả khi người dùng không nhắc.
---

# Go backend standard

```
Request ─► Handler/DTO ─► UseCase ─► Repository ─► DB
           bind+validate   nghiệp vụ   chỉ query
           phân quyền role  quyền resource
```

## Nguyên tắc chung

- Áp dụng cho package/endpoint mới. Code cũ giữ nguyên, không refactor hàng loạt nếu không được yêu cầu.
- Dữ liệu vào UseCase phải là struct `Input` đã bind + validate (đúng kiểu, đúng khoảng giá trị). UseCase không validate lại kiểu cơ bản, chỉ kiểm tra quy tắc domain.
- Ưu tiên thư viện chuẩn (`context`, `time`, `errors`). Ngoại lệ được dùng: framework HTTP của dự án (Gin) và `go-playground/validator/v10`. Không thêm thư viện ngoài chỉ để dùng một hàm tiện ích nhỏ.

## Handler & DTO

### DTO
- Mỗi endpoint có `Request` riêng (package `dto` hoặc `delivery/http/request`).
- Handler map `Request` → `Input` của UseCase. Không truyền struct có tag JSON/HTTP thẳng xuống UseCase.

```go
type CreateOrderRequest struct {
    CustomerID int64              `json:"customer_id" binding:"required,gt=0"`
    StartDate  string             `json:"start_date"  binding:"omitempty,datetime=2006-01-02"`
    EndDate    string             `json:"end_date"    binding:"omitempty,datetime=2006-01-02"`
    Items      []OrderItemRequest `json:"items"       binding:"required,min=1,dive"`
}

type OrderItemRequest struct {
    ProductID int64 `json:"product_id" binding:"required,gt=0"`
    Quantity  int   `json:"quantity"   binding:"required,min=1"`
}
```

### Bind & validate
- Chỉ dùng MỘT cơ chế: `c.ShouldBindJSON` / `c.ShouldBindQuery` với tag `binding` (Gin đã dùng validator v10). Không gọi thêm `validate.Struct` cho cùng struct.
- Lỗi validate được dịch sang tiếng Việt qua một translator tập trung (một chỗ duy nhất), trả dạng `[{field, rule, message}]`.
- Trường lọc ngày chỉ nhận `YYYY-MM-DD`. Timezone nạp một lần (`time.LoadLocation("Asia/Ho_Chi_Minh")`), không gọi lại mỗi request.

### Lọc theo ngày (nửa mở)
DTO có hàm đổi sang `time.Time`:
- `from` = `00:00:00` của ngày bắt đầu.
- `to` = `00:00:00` của **ngày kế tiếp** ngày kết thúc (cận trên loại trừ).

Repository lọc `>= @from AND < @to`. Không dùng `23:59:59` để tránh mất bản ghi có phần dưới giây.

### Phân quyền
- Phân quyền theo role/token: ở Middleware hoặc đầu Handler. Không đủ quyền → `403`; chưa đăng nhập/token sai → `401`.
- Phân quyền theo resource (ví dụ "chỉ chủ bài viết được sửa") cần dữ liệu DB, nên kiểm tra trong UseCase và trả `ErrForbidden`.

### Response & điều phối
- Handler chỉ: lấy `c.Request.Context()`, bind, gọi UseCase, map kết quả sang JSON. Không viết logic tính toán, không truy vấn SQL/ORM, không phụ thuộc Repository.
- Truyền `c.Request.Context()` xuống mọi lời gọi để hủy tác vụ khi client ngắt kết nối.

```json
{ "status": true, "message": "Thành công", "data": {}, "error_code": "" }
```

- Thành công: HTTP 200, `status: true`, payload trong `data`.
- Lỗi: HTTP tương ứng, `status: false`, `data: null`, `message` tiếng Việt, `error_code` cho client rẽ nhánh.
- Bảng map `Code → HTTP status` nằm ở tầng Handler (một chỗ duy nhất).

| Tình huống | HTTP |
|---|---|
| Dữ liệu sai / vi phạm quy tắc domain | 400 (hoặc 422) |
| Chưa đăng nhập / token sai | 401 |
| Không đủ quyền | 403 |
| Không tìm thấy | 404 |
| Xung đột (trùng, sai trạng thái) | 409 |
| Quá giới hạn request | 429 |
| Lỗi hệ thống | 500 |

## UseCase / Service

- Chứa toàn bộ tính toán, điều kiện domain, orchestration, transaction, gọi external client.
- Mọi hàm nhận `ctx context.Context` đầu tiên. Không import `gin`, `net/http`.
- Có thể đặt timeout cho tác vụ ngoài bằng `context.WithTimeout`.

```go
func (uc *orderUseCase) CreateOrder(ctx context.Context, in dto.CreateOrderInput) (*domain.Order, error)
```

### Lỗi
- Không dùng `panic` thay cho error nghiệp vụ.
- Không để raw DB error (`sql.ErrNoRows`, deadlock, lỗi driver) lọt lên Handler.
- `AppError` **không chứa HTTP status**:

```go
type AppError struct {
    Code    string // "ORDER_NOT_FOUND"
    Message string // tiếng Việt thân thiện cho client
    RawErr  error  // lỗi gốc, chỉ để log, không trả client
}

func (e *AppError) Error() string { return e.Code }
func (e *AppError) Unwrap() error { return e.RawErr }
```

- Dùng sentinel/`AppError` trong `domain` (`ErrOrderNotFound`, `ErrInvalidStatus`, ...) và kiểm tra bằng `errors.Is` / `errors.As`. Bọc ngữ cảnh bằng `fmt.Errorf("create order: %w", err)`.
- Phân luồng:
  - Không tồn tại → NotFound, message "Không tìm thấy dữ liệu".
  - Vi phạm domain → Invalid/Conflict, message tiếng Việt nêu rõ lý do.
  - Lỗi hệ thống → trả `AppError` chung "Hệ thống gặp sự cố, vui lòng thử lại sau", giữ `RawErr`.
- **Log đúng một lần**, ở Handler hoặc middleware, khi gặp lỗi hệ thống (kèm request ID). UseCase/Repo không log lại cùng lỗi mà chỉ bọc và trả lên.

### Transaction
- UseCase quyết định ranh giới transaction thông qua một `TxManager` (hoặc tương đương) có dạng `WithTx(ctx, func(ctx context.Context) error) error`. Repository lấy tx từ `ctx` nếu có, nếu không thì dùng kết nối thường. Không truyền `*sql.Tx` qua tham số từng hàm.

## Repository

- Mọi method nhận `ctx context.Context` đầu tiên và chỉ query, map row → entity, trả error.
- Map `sql.ErrNoRows` → `domain.ErrNotFound` (hoặc sentinel tương ứng) ngay tại Repository. Với lệnh update/delete, kiểm tra `RowsAffected` khi cần.
- Không validate, không logic nghiệp vụ, không ép kiểu từ chuỗi chưa chuẩn hóa.
- Với `rows`: `defer rows.Close()`, và kiểm tra `rows.Err()` sau vòng `rows.Next()`.
- Danh sách trả về nhiều bản ghi phải có phân trang (`limit`/`offset` hoặc cursor) trừ khi tập dữ liệu chắc chắn nhỏ.

## Kiến trúc Go

- Interface đặt ở nơi tiêu thụ (UseCase khai báo interface Repo nó cần; Handler khai báo interface UseCase nó cần). Hàm khởi tạo trả concrete type.
- YAGNI: chỉ tạo interface khi cần mock để unit test UseCase hoặc có từ 2 implementation trở lên. Struct chỉ dùng một nơi, không test mock thì không cần interface.
- Ưu tiên composition (struct embedding), tránh abstract layer thừa.
- Không spawn `go func()` thiếu kiểm soát vòng đời (cần `WaitGroup`/`errgroup`, channel đóng hoặc context timeout).
- Không copy giá trị `sync.Mutex`/`sync.WaitGroup`; luôn dùng và truyền bằng pointer.

## Checklist

- [ ] Input đã bind + validate (tag `binding`) trước khi gọi UseCase; Handler map `Request` → `Input`.
- [ ] Quyền role kiểm tra ở Middleware/Handler; quyền resource kiểm tra ở UseCase.
- [ ] UseCase và Repository nhận `ctx` đầu tiên; Handler truyền `c.Request.Context()`.
- [ ] Handler không có query DB, không có logic domain, không gọi thẳng Repository.
- [ ] UseCase không import Gin/`net/http`; `AppError` không chứa HTTP status.
- [ ] Không có `panic` thay control flow.
- [ ] Response đúng `{status, message, data, error_code}`, message tiếng Việt, không lộ SQL/raw error.
- [ ] `ErrNoRows` được map sang lỗi domain; `rows.Close()` và `rows.Err()` được xử lý.
- [ ] Lọc ngày dùng `>= from AND < to` (ngày kế tiếp 00:00).
- [ ] Lỗi hệ thống được log đúng một lần.
- [ ] `golangci-lint run` và `go test -race ./...` pass.
