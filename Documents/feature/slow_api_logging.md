# Tài Liệu Kỹ Thuật: Ghi Log API Chậm (Slow API Logging)

Tài liệu này mô tả cơ chế phát hiện và ghi lại các request xử lý chậm ra một file log riêng, phục vụ việc theo dõi hiệu năng và tìm điểm cần tối ưu của hệ thống.

---

## 1. Mục Đích

Log request tiêu chuẩn của Chi in ra stdout tất cả mọi request, nên khi hệ thống chạy lâu ngày rất khó lọc ra những API thực sự gây chậm cho người dùng. Tính năng này tách riêng phần dữ liệu đáng quan tâm:

- Chỉ ghi các request có thời gian xử lý **vượt ngưỡng** (mặc định 1 giây).
- Ghi dạng **JSON mỗi dòng một bản ghi** để dễ dùng `jq`, `grep` hoặc đẩy vào công cụ phân tích log.
- File log **tự tách theo ngày** và **tự xóa file quá hạn**, không cần cài thêm logrotate.

---

## 2. Cấu Hình (.env)

Cả ba biến đều **tùy chọn**. Nếu không khai báo trong `.env` (hoặc khai nhưng để trống), hệ thống tự dùng giá trị mặc định.

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `SLOW_API_THRESHOLD` | `1000` | Ngưỡng tính bằng **mili giây**. Request có thời gian xử lý ≥ ngưỡng này sẽ bị ghi log. Đặt `0` để **tắt hẳn** tính năng. |
| `SLOW_API_LOG_FILE` | `logs/slow_api.log` | Đường dẫn file gốc. Hệ thống tự chèn ngày vào tên file khi ghi (xem mục 4). Đường dẫn tương đối được tính từ thư mục chạy binary. |
| `SLOW_API_LOG_MAX_DAYS` | `7` | Số ngày log được giữ lại. Các file cũ hơn bị xóa tự động khi sang ngày mới. |

Ví dụ khai báo trong `.env`:

```bash
SLOW_API_THRESHOLD=1000
SLOW_API_LOG_FILE=logs/slow_api.log
SLOW_API_LOG_MAX_DAYS=7
```

**Lưu ý:** giá trị sai kiểu (ví dụ `SLOW_API_THRESHOLD=abc`, số âm, hoặc `SLOW_API_LOG_MAX_DAYS=0`) sẽ làm `config.Load()` trả lỗi và server **không khởi động** — chủ ý để một typo không âm thầm bị bỏ qua khiến bạn tưởng đã đổi ngưỡng.

---

## 3. Kiến Trúc & Vị Trí Code

| File | Vai trò |
|---|---|
| `internal/router/slow_api_middleware.go` | Middleware đo thời gian mỗi request, thu thập thông tin request/response khi vượt ngưỡng. |
| `internal/service/logger/slow_api.go` | `SlowAPILogger`: dựng bản ghi JSON, ghi xuống file, xoay file theo ngày và dọn file cũ. |
| `config/config.go` | Đọc ba biến môi trường ở trên (`getSlowAPISettings`). |
| `internal/router/options.go` | Option `WithSlowAPILogger` để truyền logger vào router. |
| `cmd/api/main.go` | Khởi tạo logger từ config và đóng file khi shutdown. |

**Thứ tự middleware** trong `internal/router/router.go`:

```go
r.Use(recoverMiddleware)
r.Use(requestLogger(cfg.trustedProxies))
r.Use(slowAPILogger(cfg.slowAPILogger))
```

`slowAPILogger` đặt **sau** `requestLogger` để lấy được Client IP thật đã được resolve từ trusted proxy (Cloudflare Tunnel / Nginx) và lưu trong context, thay vì IP của proxy nội bộ.

**Luồng hoạt động:**

1. Middleware ghi mốc `start := time.Now()` và bọc `ResponseWriter` bằng `middleware.WrapResponseWriter` của Chi để lấy status code và số byte đã ghi.
2. Handler chạy bình thường.
3. Trong `defer`, tính `time.Since(start)`. Nếu **nhỏ hơn ngưỡng thì dừng ngay** — đường nóng không tốn thêm gì ngoài một phép trừ thời gian.
4. Nếu vượt ngưỡng: gom thông tin request (route pattern, query, IP, user agent...), đọc số liệu heap rồi ghi một dòng JSON.
5. Lỗi khi ghi file **không làm hỏng request**, chỉ in cảnh báo ra stdout.

Khi `SLOW_API_THRESHOLD=0`, `NewSlowAPILogger` trả về `nil` và middleware trở thành pass-through (`return next`), không phát sinh chi phí nào.

---

## 4. File Log & Cơ Chế Xoay Vòng

Đường dẫn khai báo là `logs/slow_api.log`, nhưng file thực tế được chèn ngày vào trước phần mở rộng:

```
logs/slow_api-2026-09-18.log
logs/slow_api-2026-09-17.log
logs/slow_api-2026-09-16.log
```

- Thư mục `logs/` được tạo tự động ở lần ghi đầu tiên; nếu cả ngày không có request chậm thì không sinh file nào.
- Khi sang ngày mới, file cũ được đóng, file mới được mở và các file vượt quá `SLOW_API_LOG_MAX_DAYS` bị xóa.
- Thư mục `logs/` đã nằm ngoài phạm vi theo dõi của Git (`.gitignore` chỉ whitelist các đuôi mã nguồn), không cần cấu hình thêm.

---

## 5. Định Dạng Bản Ghi

Mỗi request chậm là **một dòng JSON**:

```json
{"message":"Slow API detected","context":{"method":"GET","uri":"/api/v1/invoice?page=2","path":"/api/v1/invoice","route":"/api/v1/invoice/","query_params":"page=2","duration_ms":4067.14,"status_code":200,"client_ip":"113.161.50.20","user_agent":"Mozilla/5.0","request_size":0,"response_size":3456,"heap_alloc_bytes":12582912,"heap_sys_bytes":25165824,"goroutines":18,"timestamp":"2026-09-18 16:33:13"},"level_name":"WARNING","channel":"slow_api","datetime":"2026-09-18T16:33:13.677+07:00"}
```

| Trường | Ý nghĩa |
|---|---|
| `method`, `uri`, `path`, `query_params` | Thông tin request. `uri` kèm query string, `path` đã bỏ query. |
| `route` | **Route pattern** của Chi, ví dụ `/api/v1/invoice/{id}`. Dùng để gom nhóm thống kê theo endpoint thay vì theo từng ID cụ thể. |
| `duration_ms` | Thời gian xử lý, làm tròn 2 chữ số thập phân. |
| `status_code`, `response_size` | Status và số byte body đã trả về. |
| `request_size` | `Content-Length` của request (`-1` nếu client không khai). |
| `client_ip` | IP thật của client (đã resolve qua trusted proxy). |
| `user_agent` | Header `User-Agent`. |
| `heap_alloc_bytes`, `heap_sys_bytes`, `goroutines` | Ảnh chụp bộ nhớ và số goroutine **của cả tiến trình** tại thời điểm request kết thúc. |
| `timestamp`, `datetime` | Thời điểm request bắt đầu (dạng dễ đọc và dạng RFC3339). |

**Lưu ý về số liệu bộ nhớ:** Go không đo được bộ nhớ theo từng request (nhiều request chạy song song trên cùng một heap), nên đây là snapshot toàn tiến trình, chỉ dùng để nhận biết áp lực bộ nhớ khi API chậm — **không** phải lượng RAM riêng của request đó. `runtime.ReadMemStats` chỉ được gọi khi request đã chậm nên không ảnh hưởng hiệu năng đường nóng.

---

## 6. Cách Đọc & Phân Tích Log

Xem các request chậm gần nhất:

```bash
tail -n 20 logs/slow_api-$(date +%F).log | jq '.context | {duration_ms, method, uri, status_code}'
```

Top 10 endpoint chậm nhất trong ngày (gom theo route pattern):

```bash
jq -r '.context | "\(.duration_ms)\t\(.method) \(.route)"' logs/slow_api-$(date +%F).log \
  | sort -rn | head -10
```

Đếm số lần chậm theo endpoint để tìm điểm cần tối ưu trước:

```bash
jq -r '.context | "\(.method) \(.route)"' logs/slow_api-*.log | sort | uniq -c | sort -rn
```

Lọc riêng các request cực chậm (trên 5 giây):

```bash
jq 'select(.context.duration_ms > 5000) | .context' logs/slow_api-*.log
```

---

## 7. Quy Trình Sử Dụng Khi Tối Ưu Hiệu Năng

1. Chạy hệ thống với ngưỡng mặc định 1000ms trong vài ngày để có dữ liệu thật.
2. Dùng lệnh ở mục 6 để tìm endpoint xuất hiện nhiều nhất — đó thường là điểm đáng tối ưu trước, không nhất thiết là endpoint chậm nhất.
3. Với mỗi endpoint: kiểm tra truy vấn N+1, thiếu index, hoặc gọi API bên thứ ba (Zalo, PayOS, SePay) đang chạy đồng bộ trong request.
4. Sau khi tối ưu, **hạ ngưỡng** (ví dụ `SLOW_API_THRESHOLD=500`) để tiếp tục phát hiện lớp vấn đề tiếp theo.

---

## 8. Kiểm Thử (Tests)

| File | Nội dung kiểm thử |
|---|---|
| `internal/service/logger/slow_api_test.go` | Shape của JSON, làm tròn `duration_ms`, xoay file theo ngày và xóa file quá hạn, an toàn khi logger `nil`. |
| `internal/router/slow_api_middleware_test.go` | Chỉ ghi log request vượt ngưỡng, lấy đúng route pattern/query/status/response size, pass-through khi tính năng tắt. |
| `config/config_test.go` | Giá trị mặc định khi biến không khai báo / để trống / chỉ có khoảng trắng, giá trị tùy chỉnh, và báo lỗi khi giá trị sai. |

```bash
go test ./config/ ./internal/router/ ./internal/service/logger/
```
