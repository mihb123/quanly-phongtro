# Logic Chọn Nhà Trọ & Phòng Tự Động Khi Tạo Hóa Đơn

Tài liệu này mô tả chi tiết logic tự động chọn nhà trọ (house) trong các tính năng tạo hóa đơn (bao gồm tạo hóa đơn đơn lẻ và tạo hóa đơn nhanh cho nhiều phòng). 

## Mục đích

Khi hệ thống có nhiều nhà trọ, việc tự động chọn đúng nhà trọ mà admin đang cần thao tác sẽ tiết kiệm thời gian đáng kể.
Mục tiêu của logic này là giúp admin tiết kiệm thời gian bằng cách:
1. Ghi nhớ sự lựa chọn gần nhất của admin về nhà trọ.
2. Tự động chuyển đổi nhà trọ khi nhà trọ hiện tại đã hoàn tất việc tạo hóa đơn cho tất cả các phòng (không còn phòng trống cần hóa đơn).
3. Ưu tiên chọn nhà trọ mới nhất nếu admin chưa từng chọn nhà trọ nào trước đó.
4. Tự động gợi ý và chọn tiếp **phòng tiếp theo** chưa được tạo hóa đơn (đã được sắp xếp theo AlphaB/Tên phòng) để admin thao tác liên tục.

## Nguyên tắc hoạt động (Custom Hook: `useRecommendedHouse`)

Logic được đóng gói trong một custom hook React `useRecommendedHouse.ts`. Hook này nhận vào danh sách các nhà trọ và kỳ hóa đơn (period), sau đó trả về `recommendedHouseId`. 

Luồng xử lý (Thuật toán):
1. **Kiểm tra ngoại lệ**:
   - Nếu không có nhà trọ nào: Trả về chuỗi rỗng.
   - Nếu chỉ có 1 nhà trọ: Trả về nhà trọ duy nhất.

2. **Lấy lịch sử lựa chọn**:
   - Truy xuất `lastSelectedHouseId_Invoice` từ `localStorage`.

3. **Sắp xếp danh sách ưu tiên (Candidate Houses)**:
   - Toàn bộ danh sách nhà trọ ban đầu được sắp xếp theo thời gian tạo (`created_at`) giảm dần (nhà trọ mới nhất lên đầu).
   - Nếu tồn tại `lastSelectedHouseId` trong lịch sử và nhà trọ đó vẫn còn trong danh sách, nhà trọ này sẽ được đưa lên vị trí đầu tiên (ưu tiên cao nhất).

4. **Tìm nhà trọ phù hợp (Có phòng chưa tạo hóa đơn)**:
   - Duyệt qua danh sách ưu tiên vừa tạo.
   - Với mỗi nhà trọ, gọi API lấy danh sách các phòng (`getRoomsByHouse`).
   - Lọc ra các phòng đang có người ở (`status === 'OCCUPIED'`). Nếu không có phòng nào có người ở, bỏ qua nhà trọ này.
   - Gọi API lấy danh sách hóa đơn (`getInvoices`) của nhà trọ đó.
   - Kiểm tra xem có phòng nào đang có người ở nhưng chưa có hóa đơn trong kỳ (`period`) hiện tại hay không.
   - Nếu tìm thấy một nhà trọ có ít nhất 1 phòng chưa tạo hóa đơn, nhà trọ đó sẽ được chọn (`foundId = house.id`) và vòng lặp dừng lại.
   - Nếu duyệt hết tất cả các nhà trọ mà không tìm thấy phòng nào thiếu hóa đơn (tức là toàn bộ các nhà trọ đều đã tạo đầy đủ), thuật toán sẽ fallback về nhà trọ đầu tiên trong danh sách ưu tiên (nhà trọ vừa thao tác gần nhất, hoặc nhà trọ mới nhất).

## Áp dụng vào UI Components

### 1. `QuickCreateInvoiceModal.tsx`
- Sử dụng `useRecommendedHouse` để lấy ra `recommendedHouseId`.
- Cập nhật state `selectedHouseId` khi `recommendedHouseId` có giá trị.
- Hiển thị UI bị vô hiệu hóa (`disabled`) khi hook đang ở trạng thái `isLoading` để tránh admin thao tác trước khi việc tính toán hoàn tất.
- Khi người dùng chủ động chọn một nhà trọ khác từ dropdown, kích hoạt hàm `saveSelectedHouse(id)` để lưu lại vào `localStorage`.

### 2. `CreateInvoiceModal.tsx`
- Sử dụng `useRecommendedHouse` để lấy `recommendedHouseId` và set vào React Hook Form bằng `setValue('house_id', recommendedHouseId)`.
- Khi biến `watchHouseId` thay đổi do admin chọn từ dropdown, kích hoạt lưu lại cấu hình qua `localStorage`.
- Sử dụng thêm `useRecommendedRoom` để tự động chọn phòng kế tiếp chưa có hóa đơn. Khi nhà trọ (`house_id`) thay đổi hoặc khi lưu xong một hóa đơn (modal mở lại / trigger period thay đổi), hook này sẽ tính toán phòng chưa có hóa đơn (dựa theo thứ tự tên phòng ASC) và `setValue('room_id', recommendedRoomId)`.

## 2. Nguyên tắc hoạt động Chọn Phòng (Custom Hook: `useRecommendedRoom`)

Logic chọn phòng được đóng gói trong `useRecommendedRoom.ts`. Hook này nhận vào `houseId`, `period`, và danh sách `rooms`, trả về `recommendedRoomId`.

Luồng xử lý:
1. Lọc các phòng đang được thuê (`status === 'OCCUPIED'`).
2. Sắp xếp các phòng này theo tên tăng dần (`asc`).
3. Gọi API lấy danh sách hóa đơn của nhà trọ trong kỳ hiện tại.
4. Duyệt qua danh sách phòng đã sắp xếp, phòng đầu tiên chưa có trong danh sách hóa đơn sẽ được chọn.
5. Trường hợp tất cả các phòng đều đã có hóa đơn, fallback chọn phòng đang thuê đầu tiên (để tránh lỗi trống selection).

## Lợi ích
- Trải nghiệm liền mạch: Chặn hiện tượng phải tự tay chọn lại từ đầu khi sang tháng mới hoặc khi vừa lưu xong một hóa đơn.
- Điều hướng thông minh: Khi "Lưu tất cả" xong cho một nhà trọ, việc mở lại modal sẽ tự động nhận diện nhà trọ đó đã xong và chuyển sang nhà trọ tiếp theo. Với tạo hóa đơn đơn lẻ, tự động jump sang phòng chưa có hóa đơn theo thứ tự bảng chữ cái.
