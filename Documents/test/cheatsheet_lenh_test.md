# Cheat Sheet: Các Lệnh Test Dự Án `quanly-phongtro`

Tài liệu này tổng hợp nhanh các câu lệnh test dành cho môi trường Local (máy cá nhân). Tùy vào trạng thái của project (chưa tạo mock hay đã tạo mock), bạn chạy các lệnh tương ứng dưới đây.

---

## 1. Trường Hợp 1: CHƯA tạo Mock (Hoặc vừa sửa code Interface)

Nếu bạn vừa clone dự án về, hoặc vừa thêm/sửa một hàm trong các Interface của `internal/model` hoặc `internal/service`, bạn **bắt buộc** phải sinh lại các file mock trước khi chạy test, nếu không test sẽ báo lỗi (thiếu phương thức, kiểu dữ liệu không khớp).

### Lệnh tạo Mock toàn bộ cho dự án
Bạn cần chạy chuỗi lệnh sau tại thư mục gốc của dự án (`/media/minhchu1336/Data/quanly-phongtro`):

```bash
# 1. Tạo mock cho tầng Repository (Model)
mockgen -source=internal/model/user.go -destination=internal/mock/mock_model/user_mock.go -package=mock_model
mockgen -source=internal/model/room.go -destination=internal/mock/mock_model/room_mock.go -package=mock_model
mockgen -source=internal/model/house.go -destination=internal/mock/mock_model/house_mock.go -package=mock_model

# 2. Tạo mock cho tầng Service (Dành cho Zalo)
mockgen -source=internal/service/zalo_client.go -destination=internal/mock/mock_service/zalo_client_mock.go -package=mock_service
mockgen -source=internal/service/zalo_service.go -destination=internal/mock/mock_service/zalo_service_mock.go -package=mock_service
```

> **Giải thích**: 
> - `mockgen`: Công cụ sinh mã của bộ `go.uber.org/mock`.
> - `-source`: File code chứa Interface thật của bạn.
> - `-destination`: Đường dẫn file sinh ra (chứa Mock).
> - `-package`: Tên package của file sinh ra (ví dụ: `mock_model`).

---

## 2. Trường Hợp 2: ĐÃ tạo Mock Data thành công

Sau khi đảm bảo toàn bộ thư mục `internal/mock` đã được sinh thành công, bạn chỉ cần chạy các lệnh test sau để kiểm tra code:

### 2.1 Chạy toàn bộ Test của Zalo Bot (Service & Handler)
```bash
go test -v ./internal/security ./internal/service ./internal/handler
```
> **Giải thích**: Lệnh này sẽ chạy test chi tiết (`-v` / verbose) cho cả 3 package: `security` (mã hóa), `service` (logic nghiệp vụ chính của bot), và `handler` (API Controllers).

### 2.2 Chạy test cho một package bất kỳ (Ví dụ: service)
```bash
go test -v ./internal/service
```
> **Giải thích**: Chỉ quét và chạy các file có đuôi `_test.go` nằm trong thư mục `internal/service`.

### 2.3 Chạy Test kèm xem độ phủ (Coverage)
```bash
go test -cover ./internal/service ./internal/handler
```
> **Giải thích**: Lệnh này không in chi tiết từng hàm (`-v`), mà thay vào đó sẽ in ra **tỉ lệ %** số dòng code đã được chạy qua bởi test (coverage). Giúp bạn biết code của mình đã được test kỹ hay chưa.

### 2.4 Chạy duy nhất một test cụ thể (Debug)
```bash
go test -v -run TestZaloService_HandleWebhook ./internal/service
```
> **Giải thích**: Thay vì chạy toàn bộ, lệnh này dùng cờ `-run` khớp với tên hàm `TestZaloService_HandleWebhook` trong package `service`. Cực kỳ hữu ích khi bạn đang debug một test case bị lỗi.

### 2.5 Kiểm tra lỗi cú pháp/logic ngầm (Go Vet)
```bash
go vet ./...
```
> **Giải thích**: Trước khi chạy `go test`, chạy lệnh này sẽ kiểm tra xem bạn có đang truyền sai tham số hay khai báo thiếu phương thức mock nào không. Lệnh này quét toàn bộ dự án (`./...`) và in ra lỗi (nếu có). 
