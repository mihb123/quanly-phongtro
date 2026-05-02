# HOUSE-14: Chuẩn hóa API Response (API Response Standard)

## 1. Overview
Yêu cầu thiết lập một chuẩn chung cho mọi API response trả về Frontend. Đội ngũ Backend (Golang) cần định nghĩa 1 cấu trúc chung (Struct) và các logic tiện ích (Helper/Wrapper) để chuẩn hóa output cho mọi API, thay vì phải tự viết format JSON cho từng controller/handler riêng biệt.

## 2. Định dạng Response Chuẩn
Mọi API response (bao gồm cả thành công và thất bại) đều cần tuân theo cấu trúc JSON cơ bản sau:
```json
{
  "status": 200,
  "data": {},
  "message": "Thông báo trả về cho FE"
}
```

## 3. Cấu trúc (Struct) và Logic Helpers

Trong Golang, nên thiết lập một package `response` (hoặc tên tương tự trong thư mục `pkg/utils` hay `internal/http/response`) chứa định nghĩa file struct chung và các hàm helper tĩnh. Tùy thuộc vào HTTP Router đang sử dụng (Gin, Echo, Fiber, cơ bản net/http...), các hàm này nên nhận tham số là request context.

### Các thành phần cần định nghĩa để Dev thực hiện:

1. **Base Response Struct:** Struct định nghĩa JSON trả về với ít nhất 3 tham số `Status` (int), `Data` (interface{}), `Message` (string).
2. **Success Helper (`SuccessResponse`):** 
   - Hàm nhận tham số truyền vào: HTTP Context, Dữ liệu `data`, `message` và `status` báo thành công (200).
   - Hàm sẽ tự serialize sang struct chuẩn và ghi vào JSON response.
3. **Error Helper (`ErrorResponse`):**
   - Dùng thay thế khi bắt gặp exception hay business logic error (`err != null`).
   - Nhận vào `message` (thông báo lỗi tự định nghĩa), và `status` (mã HTTP ứng với lỗi VD: `400 Bad Request`, `401 Unauthorized`, `500 Internal Server error`).
   - Field `data` sẽ thường để null.
4. **Pagination Response Helper (`PaginatedResponse`):**
   - Viết response mở rộng: bổ sung các field về `page`, `size`, `total`, `maxPage` vào struct.
   - Giúp Frontend hiển thị mượt mà trên Grid/Table mà biết rõ tổng số trang.
5. **Validation Error Helper (`ValidationErrorResponse`):**
   - Xử lý các lỗi khi format đầu vào (body, params) của API không đúng (validation tags error).
   - Struct bao gồm field `list_errors` (mảng lưu object báo chi tiết field nào sai định dạng gì).

## 4. Cách áp dụng vào Handler

Các handler không được gọi trực tiếp method write JSON có sẵn (VD: `c.JSON(200, gin.H{...})`) với cấu trúc tự túc bên trong nữa.

**Mô tả Logic (Ví dụ pseudo-code):**
- Khi API xử lý thành công: Gọi helper trả về `response.Success(context, usersData, "Lấy danh sách user thành công", 200)`
- Khi API gặp lỗi logic/DB: Gọi helper `response.Error(context, "Lỗi khi lấy dữ liệu: " + err.Error(), 500)`
- Do vậy, code ở Handler trông sẽ rất gọn gàng và không lặp lại mã khai báo object response.

## 5. Todo List (Cho Backend Developer)
- [ ] Thiết lập package common cho standardized response model.
- [ ] Viết Struct base `APIResponse`, `PaginatedResponse`, `ValidationErrorResponse` có format tags JSON đầy đủ.
- [ ] Implement các function util helper (`Success`, `Error`, `Paginated`, `ValidationError`) nhận tham số context của HTTP framework đang dùng.
- [ ] Refactor các chức năng API hiện tại đang có (Authentication, CRUD...) để đổi response thuần sang sử dụng bộ base helper này.

## 6. Lợi ích của việc chuẩn hóa
* **Đồng nhất:** Bắt trọn 1 format giúp Frontend dễ dàng cấu hình "Global Interceptor" (ví dụ dùng Axios báo alert toast ngay khi có status lỗi) mà không tốn công parse tay ở từng trang.
* **Tái sử dụng cao & Clean code:** Tối giản mã nguồn của Controller/Handler cho dễ đọc, dễ viết test.
* **Dễ maintain:** Khi có yêu cầu thay mới response format (ví dụ thêm 1 trường "timestamp") thì chỉ cần mở file trong utils `response` update 1 lần cho toàn dự án.
