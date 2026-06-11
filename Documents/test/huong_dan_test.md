# Hướng Dẫn Chạy Và Viết Test Trong Dự Án (Unit Test & Mocking)

Chào mừng bạn đến với tài liệu hướng dẫn kiểm thử (testing) của dự án Quản Lý Phòng Trọ. Hệ thống của chúng ta ưu tiên sử dụng **Unit Test thuần túy**, kết hợp với công cụ **Mocking**, nhằm đảm bảo các bài test có thể chạy cực kỳ nhanh và hoạt động độc lập ở bất kỳ môi trường nào (Local, CI/CD) mà không cần cấu hình cơ sở dữ liệu thật.

## 1. Các lệnh chạy test cơ bản

Bạn có thể sử dụng các lệnh Go tiêu chuẩn để chạy test:

- **Chạy toàn bộ test trong dự án**:
  ```bash
  go test -v ./...
  ```

- **Chạy test cho một package cụ thể** (Ví dụ: `service`):
  ```bash
  go test -v ./internal/service
  ```

- **Chạy một test function cụ thể**:
  ```bash
  go test -v -run TestZaloService_HandleWebhook ./internal/service
  ```

### Kiểm tra Test Coverage (Độ bao phủ code)

Để xem tỷ lệ code đã được kiểm thử, bạn có thể chạy:
```bash
go test -cover ./...
```

Để xuất báo cáo chi tiết ra file HTML trực quan:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 2. Làm việc với Mock (gomock)

Dự án sử dụng thư viện `go.uber.org/mock/mockgen` để tự động sinh mã giả (mock code) cho các interface (ví dụ: `UserRepository`, `ZaloClient`). Điều này giúp chúng ta test phần logic (Service/Handler) mà không cần quan tâm đến tầng Database (Repository) hay External API.

### 2.1 Cài đặt công cụ mockgen
Nếu máy bạn chưa có `mockgen`, hãy cài đặt thông qua lệnh:
```bash
go install go.uber.org/mock/mockgen@latest
```
*(Đảm bảo thư mục `$GOPATH/bin` đã được cấu hình trong biến môi trường `$PATH` của hệ điều hành).*

### 2.2 Cách sinh lại Mock Code khi thay đổi Interface
Bất cứ khi nào bạn **thêm/xóa/sửa** một phương thức (method) trong các Interface ở thư mục `internal/model` hoặc `internal/service`, bạn **BẮT BUỢC** phải sinh lại file mock.

Lệnh chạy mockgen cơ bản:
```bash
# Ví dụ sinh mock cho RoomRepository
mockgen -source=internal/model/room.go -destination=internal/mock/mock_model/room_mock.go -package=mock_model
```

**Mẹo**: Để sinh mock nhanh cho các interface thường dùng, bạn có thể chạy danh sách lệnh sau tại thư mục gốc của dự án:
```bash
mockgen -source=internal/model/user.go -destination=internal/mock/mock_model/user_mock.go -package=mock_model
mockgen -source=internal/model/room.go -destination=internal/mock/mock_model/room_mock.go -package=mock_model
mockgen -source=internal/model/house.go -destination=internal/mock/mock_model/house_mock.go -package=mock_model
mockgen -source=internal/service/zalo_client.go -destination=internal/mock/mock_service/zalo_client_mock.go -package=mock_service
mockgen -source=internal/service/zalo_service.go -destination=internal/mock/mock_service/zalo_service_mock.go -package=mock_service
```

---

## 3. Best Practices (Thực hành tốt nhất) khi viết Test

Dưới đây là cấu trúc tiêu chuẩn khi bạn viết một Unit Test cho Service hoặc Handler:

1. **Khởi tạo Gomock Controller**:
   ```go
   ctrl := gomock.NewController(t)
   defer ctrl.Finish() // Rất quan trọng, giúp kiểm tra xem tất cả các EXPECT() đã được gọi chưa
   ```

2. **Khởi tạo các đối tượng Mock**:
   ```go
   userRepo := mock_model.NewMockUserRepository(ctrl)
   roomRepo := mock_model.NewMockRoomRepository(ctrl)
   ```

3. **Truyền Mock vào Constructor**:
   ```go
   svc, err := service.NewZaloService(client, userRepo, roomRepo, ...)
   ```

4. **Thiết lập kỳ vọng (Expectations)**: Bạn phải khai báo chính xác hàm nào của mock sẽ được gọi, tham số là gì và kết quả trả về là gì.
   ```go
   // Mong đợi hàm GetByUserID được gọi với tham số là managerID,
   // và chỉ định mock object trả về một cấu trúc giả định.
   userRepo.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{
       ID: "manager-1",
   }, nil)
   
   // Dùng gomock.Any() nếu bạn không muốn kiểm tra chính xác giá trị của một tham số
   roomRepo.EXPECT().UpdateRoom(ctx, "room-1", "house-1", gomock.Any()).Return(nil)
   ```

5. **Lưu ý quan trọng về Expectation**: 
   - Nếu code của bạn gọi một hàm của Mock mà chưa được `EXPECT()`, test sẽ **Báo Lỗi (Fail)**.
   - Nếu bạn có khai báo `EXPECT()` nhưng code thực tế không hề gọi đến hàm đó, bài test cũng sẽ **Báo Lỗi**.

Hãy luôn giữ cho logic Test đơn giản, đi thẳng vào vấn đề và đảm bảo bao phủ cả những trường hợp Happy Path (thành công) lẫn Error Path (báo lỗi). Chúc bạn code vui vẻ!
