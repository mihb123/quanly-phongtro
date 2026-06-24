# Logic tạo hóa đơn qua Zalo Chat

Bản tóm tắt logic tạo/cập nhật hóa đơn bằng tin nhắn Zalo Bot.

## 1. File chính

- `zalo_service.go`: nhận webhook, verify secret, route invoice command.
- `zalo_command_parser.go`: parse cú pháp, normalize tiếng Việt, match tên phòng.
- `zalo_invoice_command_service.go`: xử lý invoice command, pending state, phản hồi chat.
- `pending_invoice_update.go`: model pending command.
- `pending_invoice_update_repository.go`: lưu pending command.
- `zalo_invoice_delivery.go`: gửi ảnh hóa đơn sau khi invoice hoàn tất.

## 2. Cú pháp

Một phòng, dùng trong group phòng hoặc private chat tenant:

```text
#dien <chỉ số mới>
#nuoc <chỉ số mới>
```

Nhiều phòng, dùng trong private chat manager:

```text
#dien <house_code>
P101 750
P201 900
```

Reply command khi bot đang chờ phản hồi:

```text
#ok       xác nhận ghi đè
#huy      hủy pending
#1..#12   chọn tháng
```

Rule: reply command phải có `#`. Tin nhắn `ok`, `huy`, `6` là tin nhắn thường, không phải command.

## 3. Parser

`ParseCommand`:

- Trim, lowercase, bỏ dấu tiếng Việt.
- Nhận `#dien`, `#nuoc`, `#ok`, `#huy`, `#1..#12`.
- Một dòng là single command.
- Nhiều dòng là batch command.
- `#dien abc` một dòng được hiểu là batch command thiếu entries với `house_code=abc`.

`MatchRoom` normalize tên phòng:

- Bỏ dấu, lowercase.
- Bỏ khoảng trắng, `-`, `_`.
- Bỏ prefix `phong` hoặc `p`.
- Giữ chữ cái trong mã phòng.

Ví dụ:

- `P101`, `Phòng 101`, `phong101` -> `101`.
- `A101`, `Phòng A-101` -> `a101`.
- `A101` và `B101` không bị gộp thành `101`.

## 4. Entry point

`HandleWebhook`:

1. Verify webhook secret.
2. Xử lý link Zalo user ID nếu manager chưa link.
3. Nếu text là invoice command hoặc chat đang có pending, gọi `HandleInvoiceCommand`.
4. Nếu không, chạy các flow Zalo khác như link số điện thoại, xử lý ảnh giao dịch, auto-link group.

`commandChatID`: ưu tiên `replyChatID`, sau đó `chatID`, cuối cùng `senderID`.

## 5. Resolve kỳ hóa đơn

`resolvePeriod(roomID, forcedPeriod)`:

| Điều kiện | Kỳ được dùng |
|---|---|
| Có `forcedPeriod` | Dùng luôn `forcedPeriod` |
| Phòng chưa có invoice | Hỏi chọn tháng trước hoặc tháng hiện tại |
| Latest invoice >= tháng hiện tại | Dùng latest invoice period |
| Latest invoice < tháng hiện tại | Dùng tháng kế tiếp của latest invoice |

Ví dụ ngày `2026-06-14`:

- Chưa có invoice -> hỏi `#5` hoặc `#6`.
- Latest `2026-05` -> dùng `2026-06`.
- Pending lưu `period=2026-05` -> dùng `2026-05`.

## 6. Pending state

Pending lưu theo `manager_id + chat_id`.

| Action | Khi nào tạo | Reply hợp lệ |
|---|---|---|
| `AWAIT_PERIOD` | Chưa biết kỳ hóa đơn | `#1..#12`, `#huy` |
| `CONFIRM_OVERWRITE` | Cần xác nhận ghi đè chỉ số | `#ok`, `#huy` |
| `AWAIT_UTILITY` | Đã ghi một utility, còn thiếu utility còn lại | `#dien/#nuoc`, `#huy` |

Khi tạo pending mới, service xóa pending cũ của chat rồi tạo pending mới. Pending hiện chưa có TTL.

## 7. Flow single command

Dùng cho group phòng hoặc private chat tenant.

1. Resolve phòng từ group hoặc tenant.
2. Kiểm tra utility usage có chỉ số mới chưa.
3. Resolve kỳ hóa đơn.
4. Nếu cần chọn kỳ, tạo `AWAIT_PERIOD`.
5. Nếu cần ghi đè, tạo `CONFIRM_OVERWRITE`.
6. Build invoice input từ previous/existing invoice.
7. Gọi `InvoiceService.CreateInvoice`.
8. Nếu thiếu utility còn lại, tạo `AWAIT_UTILITY`.
9. Nếu đủ dữ liệu, gửi ảnh hóa đơn.

Manager private chat không dùng single command; manager dùng batch command.

## 8. Flow batch command

Dùng cho private chat manager.

1. Verify sender là manager đã link Zalo.
2. Tìm nhà bằng `house_code`.
3. Match từng dòng phòng.
4. Resolve kỳ từng phòng.
5. Nếu có phòng cần chọn kỳ hoặc ghi đè, tạo pending và dừng hỏi manager.
6. Nếu không, update từng phòng.
7. Trả summary; lỗi từng phòng được gom vào cuối message.

## 9. Build invoice input

`buildInvoiceInput` giữ dữ liệu cũ:

- Previous invoice: old index kỳ mới lấy từ new index kỳ trước.
- Existing invoice: giữ old/new index, other fee, discount, vehicle count, tenant count.
- Chỉ thay utility user đang nhập.

Vì vậy `#dien 750` không làm mất số nước đã có.

## 10. Ghi đè chỉ số

Hỏi `#ok/#huy` khi utility usage đã có chỉ số khác old index và khác chỉ số mới user gửi.

```text
Số điện tháng 06/2026 đã được ghi là 750.
Bạn muốn cập nhật thành 760?
Nhắn '#ok' để xác nhận, '#huy' để hủy.
```

`#ok` resume với `allowOverwrite=true`. `#huy` xóa pending và hủy thao tác.

## 11. Bổ sung utility còn thiếu

Khi user nhập một utility nhưng utility còn lại chưa xác nhận, service vẫn tạo invoice thật và lưu `AWAIT_UTILITY`.

Ví dụ:

```text
User: #dien 500
Bot: Vui lòng bổ sung số nước tháng 05/2026 bằng cú pháp: #nuoc <số mới> hoặc #huy để hủy.
```

Pending lưu tối thiểu:

```json
{"utility_type":"nuoc","room_id":"room-id","period":"2026-05"}
```

Rule quan trọng:

- User gửi đúng utility còn thiếu -> dùng `period` trong pending làm `forcedPeriod`.
- Không resolve kỳ mới trong `AWAIT_UTILITY`.
- User gửi sai utility -> nhắc utility đang chờ, giữ pending.
- User gửi thiếu chỉ số -> báo lỗi, giữ pending để retry.

Ví dụ giữ đúng kỳ:

```text
Ngày 14/06/2026
#dien 500 -> chọn #5 -> chờ #nuoc
#nuoc 120 -> update invoice 2026-05, không nhảy sang 2026-06
```

## 12. Zero usage, fixed billing, gửi ảnh

Zero usage:

- Nếu điện/nước không tăng, user vẫn nhập lại số cũ để xác nhận.
- Ví dụ nước cũ `120`, tháng này không dùng nước: `#nuoc 120`.

Fixed billing:

- Utility `FIXED` không cần chỉ số mới.
- Missing utility check bỏ qua utility fixed.

Gửi ảnh:

- Invoice hoàn tất thì generate ảnh và gửi qua Zalo.
- Nếu có `group_chat_id`, gửi vào group.
- Nếu không có group, gửi manager linked Zalo và tenant linked Zalo.

## 13. Rule maintain cần giữ

- Reply command phải có `#`.
- `AWAIT_UTILITY` luôn dùng `period` trong pending.
- Không xóa `AWAIT_UTILITY` khi user nhập sai utility hoặc thiếu chỉ số.
- Không bỏ chữ cái khi normalize phòng.
- Không đổi logic zero usage nếu nghiệp vụ vẫn yêu cầu user nhập lại số cũ.
- Thêm pending action mới thì cập nhật `handlePendingCommand` và test tương ứng.
