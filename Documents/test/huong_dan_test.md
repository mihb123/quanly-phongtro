# Hướng Dẫn Chạy Và Viết Test Trong Dự Án (Unit Test & Mocking)

Chào mừng bạn đến với tài liệu hướng dẫn kiểm thử (testing) của dự án Quản Lý Phòng Trọ. Hệ thống của chúng ta ưu tiên sử dụng **Unit Test thuần túy**, kết hợp với công cụ **Mocking**, nhằm đảm bảo các bài test có thể chạy cực kỳ nhanh và hoạt động độc lập ở bất kỳ môi trường nào (Local, CI/CD) mà không cần cấu hình cơ sở dữ liệu thật.

Tài liệu này ưu tiên hướng dẫn bạn cách chạy các bài test hiện có, sau đó là cách viết các bài test mới.

---

## 1. Yêu cầu bắt buộc trước khi chạy Test (Pre-requisites)

Vì hệ thống sử dụng Mock (mã giả) để test, nếu bạn vừa kéo code mới về hoặc có sự thay đổi trong cấu trúc Interface, bạn **bắt buộc** phải sinh lại mã mock trước khi chạy test để tránh lỗi biên dịch.

Hãy chạy tuần tự các lệnh sau tại thư mục gốc của dự án:

1. **Tải các thư viện phụ thuộc (nếu chưa có)**:
   ```bash
   go mod tidy
   ```

2. **Cài đặt công cụ sinh Mock (chỉ cần làm 1 lần)**:
   ```bash
   go install go.uber.org/mock/mockgen@latest
   ```
   *(Đảm bảo thư mục `$GOPATH/bin` đã được cấu hình trong biến môi trường `$PATH` của hệ điều hành).*

3. **Sinh lại toàn bộ Mock code**:
   ```bash
   make mocks
   ```
   *Lệnh này sẽ tự động tìm các interface đã định nghĩa trong Makefile và sinh ra/cập nhật code mock tương ứng vào thư mục `internal/mock/`.*

---

## 2. Các lệnh chạy test cơ bản

Sau khi đã đảm bảo Mock code được cập nhật, bạn có thể sử dụng các lệnh Go tiêu chuẩn để chạy test:

- **Chạy toàn bộ test trong dự án**:
  ```bash
  go test -v ./...
  ```

- **Chạy test cho một package cụ thể** (Ví dụ: package `service`):
  ```bash
  go test -v ./internal/service
  ```

- **Chạy một test function cụ thể** (Ví dụ test hàm `HandleWebhook`):
  ```bash
  go test -v -run TestZaloService_HandleWebhook ./internal/service
  ```

### Kiểm tra Test Coverage (Độ bao phủ code)

Để xem tỷ lệ code đã được kiểm thử, bạn có thể chạy:
```bash
go test -cover ./...
```

Để xuất báo cáo chi tiết ra file HTML trực quan, giúp bạn biết chính xác dòng code nào chưa được test:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 3. Best Practices (Thực hành tốt nhất) khi viết Test

Dự án áp dụng mẫu thiết kế **Setup Struct + Sub-tests (t.Run)** kết hợp với **Table-Driven Tests**. Đây là cấu trúc tiêu chuẩn khi bạn viết một Unit Test cho Service, giúp mã dễ tái sử dụng và dọn dẹp hơn so với việc khởi tạo thủ công từng mock trong mỗi hàm.

### 3.1 Khai báo Struct chứa Mock và Setup function

Thay vì khởi tạo Mock lặp đi lặp lại trong mỗi Test function, hãy gom chúng vào một struct và tạo một hàm `setup...Test`:

```go
type mockStruct struct {
	ctrl        *gomock.Controller
	userRepo    *mock_model.MockUserRepository
	roomRepo    *mock_model.MockRoomRepository
	svc         service.ZaloService // Service thực tế cần test
}

func setupTest(t *testing.T) *mockStruct {
	ctrl := gomock.NewController(t)
	m := &mockStruct{
		ctrl:       ctrl,
		userRepo:   mock_model.NewMockUserRepository(ctrl),
		roomRepo:   mock_model.NewMockRoomRepository(ctrl),
	}

	// Khởi tạo service thật với các mock dependencies
	svc, _ := service.NewZaloService(m.userRepo, m.roomRepo)
	m.svc = svc
	
	return m
}
```

### 3.2 Cấu trúc hàm Test tiêu chuẩn

Sử dụng hàm setup bên trên kết hợp với `t.Run()` để phân chia các test case rõ ràng. Chú ý luôn gọi `defer m.ctrl.Finish()` ngay sau khi lấy mock để đảm bảo Gomock kiểm tra EXPECT():

```go
func TestService_DoSomething(t *testing.T) {
	ctx := context.Background()

	t.Run("success_case", func(t *testing.T) {
		m := setupTest(t)
		defer m.ctrl.Finish() // Bắt buộc để kiểm tra các EXPECT()

		// 1. Thiết lập kỳ vọng (Expectations)
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ID: "m1"}, nil)
		m.roomRepo.EXPECT().UpdateRoom(ctx, "r1", "h1", gomock.Any()).Return(nil)

		// 2. Thực thi hàm cần test
		err := m.svc.DoSomething(ctx, "m1", "r1", "h1")

		// 3. Kiểm tra kết quả
		assert.NoError(t, err)
	})

	t.Run("error_case_when_user_not_found", func(t *testing.T) {
		m := setupTest(t)
		defer m.ctrl.Finish()

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("not found"))

		err := m.svc.DoSomething(ctx, "m1", "r1", "h1")
		assert.Error(t, err)
	})
}
```

### 3.3 Sử dụng Table-Driven Tests cho nhiều trường hợp

Với những hàm có nhiều nhánh logic rẽ nhánh, hãy dùng mảng struct để khai báo các test case, từ đó chỉ thực thi thân hàm test một lần trong vòng lặp `for`:

```go
func TestService_ComplexLogic(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(m *mockStruct)
		wantErr bool
	}{
		{
			name: "success",
			setup: func(m *mockStruct) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
			},
			wantErr: false,
		},
		{
			name: "db error",
			setup: func(m *mockStruct) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupTest(t)
			defer m.ctrl.Finish()

			tt.setup(m) // Thực thi thiết lập mock cho case cụ thể này

			err := m.svc.ComplexLogic(ctx, "m1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

### 3.4 Lưu ý quan trọng về Expectation:
- **Missing Call**: Nếu code của bạn gọi một hàm của Mock mà chưa được `EXPECT()`, test sẽ **Báo Lỗi (Fail)**.
- **Unused Call**: Nếu bạn có khai báo `EXPECT()` nhưng code thực tế không hề gọi đến hàm đó, bài test cũng sẽ **Báo Lỗi** (do thiếu gọi hàm nên `defer ctrl.Finish()` sẽ báo lỗi).
- **Match Any**: Đối với tham số không xác định hoặc linh động, bạn có thể sử dụng `gomock.Any()`.

Hãy luôn giữ cho logic Test đơn giản, đi thẳng vào vấn đề và đảm bảo bao phủ cả những trường hợp Happy Path (thành công) lẫn Error Path (báo lỗi). Chúc bạn code vui vẻ!
