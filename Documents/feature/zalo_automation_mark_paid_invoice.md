# Logic tạo hóa đơn qua Zalo Chat

Bản tóm tắt logic tạo/cập nhật hóa đơn bằng tin nhắn Zalo Bot.

## 1. File chính

- `zalo_service.go`: nhận webhook, verify secret, route invoice command.
- `zalo_command_parser.go`: parse cú pháp, normalize tiếng Việt, match tên phòng.
- `zalo_invoice_command_service.go`: xử lý invoice command, pending state, phản hồi chat.
- `pending_invoice_update.go`: model pending command.
- `pending_invoice_update_repository.go`: lưu pending command.
- `zalo_invoice_delivery.go`: gửi ảnh hóa đơn sau khi invoice hoàn tất.
- `zalo_help_command.go`: trả lời `#help` theo vai trò và trạng thái liên kết của người gửi.
- `zalo_manager_command.go`: lệnh manager sửa dữ liệu từ chat (`#update-tenant`, `#update-room`).

## 2. Cú pháp

Một phòng, dùng trong group phòng hoặc private chat tenant. Chat đã xác định được phòng nên không cần nhập tên nhà/phòng:

```text
#dien <chỉ số mới>
#nuoc <chỉ số mới>
```

Một phòng có nêu tên phòng, dùng cho manager (private chat hoặc group):

```text
#dien <house_code> <room_name> <chỉ số mới>
#dien <room_name> <chỉ số mới>
```

- Bỏ được `house_code` khi manager chỉ quản lý đúng 1 nhà.
- `house_code` không khớp nhà nào sẽ được thử lại như từ đầu của tên phòng nhiều từ (`#dien Phòng 201 661`).
- Tenant cũng dùng được cú pháp này, nhưng chỉ với đúng phòng mình đang thuê.

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

Trợ giúp, dùng được ở mọi chat và mọi trạng thái liên kết:

```text
#help     xem danh sách lệnh (alias: #trợ giúp)
```

Lệnh sửa dữ liệu, chỉ quản lý dùng được:

```text
#update-tenant <house_code> <room_name> <số điện thoại>
#update-room   <house_code> <room_name> <group_chat_id>
#update-room   <house_code> <room_name>
```

- Room target theo đúng quy ước của `#dien`: bỏ được `house_code` khi manager chỉ có 1 nhà, bỏ được cả
  room target khi gửi trong group chat đã kết nối phòng.
- `#update-tenant` ghi đè số điện thoại hiện tại của khách thuê trong phòng đó.
- `#update-room` không kèm `group_chat_id` và gửi trong group chat sẽ kết nối chính group đó với phòng.
- Alias: `#update_tenant`, `#updatetenant`, `#update_room`, `#updateroom`.

Rule: reply command phải có `#`. Tin nhắn `ok`, `huy`, `6` là tin nhắn thường, không phải command.

## 3. Parser

`ParseCommand`:

- Trim, lowercase, bỏ dấu tiếng Việt.
- Nhận `#dien`, `#nuoc`, `#ok`, `#huy`, `#1..#12`, `#help`, `#update-tenant`, `#update-room`.
- Nhiều dòng là batch command.
- Một dòng: token cuối luôn là chỉ số mới, phần trước token cuối là nơi cần ghi.

| Một dòng | Command type | Ý nghĩa |
|---|---|---|
| `#dien 661` | `CommandUtilitySingle` | Phòng lấy từ group hoặc tenant của chat |
| `#dien P201 661` | `CommandUtilityRoom` | Phòng `P201`, nhà suy ra từ manager 1 nhà |
| `#dien 679qt P201 661` | `CommandUtilityRoom` | Nhà `679qt`, phòng `P201` |
| `#dien 679qt` | `CommandUtilityBatch` | Header batch thiếu entries |

Lệnh manager dùng cùng cách tách: token cuối là giá trị mới, phần trước là room target.

| Một dòng | Value | Room target |
|---|---|---|
| `#update-tenant 679qt P201 0912345678` | `0912345678` | nhà `679qt`, phòng `P201` |
| `#update-tenant 0912 345 678` | `0912345678` | lấy từ group chat |
| `#update-tenant P201 12345` | rỗng (số sai) | bỏ luôn, bot nhắc cú pháp |
| `#update-room 679qt P201 9876543210` | `9876543210` | nhà `679qt`, phòng `P201` |
| `#update-room 679qt P201` | rỗng -> dùng chat ID hiện tại | nhà `679qt`, phòng `P201` |

`NormalizePhone` chỉ nhận số Việt Nam 10 chữ số (`0`, `84`, `+84` + 9 chữ số), bỏ khoảng trắng, `.`, `-`,
và trả về dạng bắt đầu bằng `0`. Số điện thoại viết rời chỉ được ghép lại khi toàn bộ tham số là số
điện thoại; nếu có room target thì số phải nằm gọn trong 1 token.

`IsZaloGroupChatID` yêu cầu 6-32 chữ số. Đây là điểm phân biệt giữa `group_chat_id` và tên phòng ở cùng
vị trí cuối câu, nên tên phòng toàn số dài từ 6 chữ số sẽ bị hiểu là group ID.

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
2. Nếu text là `#help`, trả lời trợ giúp rồi dừng. Bước này đứng trước mọi flow linking nên user chưa liên kết vẫn nhận được hướng dẫn.
3. Xử lý link Zalo user ID nếu manager chưa link.
4. Nếu text là invoice command hoặc chat đang có pending, gọi `HandleInvoiceCommand`.
5. Nếu không, chạy các flow Zalo khác như link số điện thoại, xử lý ảnh giao dịch, auto-link group.

`isInvoiceCommandText` loại `#help` ra, vì help đã được trả lời trước khi tới invoice command.

`HandleInvoiceCommand` xử lý `#update-tenant`/`#update-room` trước khi đọc pending state: lệnh sửa dữ
liệu không liên quan tới hóa đơn nên không bị pending `AWAIT_PERIOD`/`CONFIRM_OVERWRITE` chặn lại, và
cũng không tạo hay xóa pending.

`commandChatID`: ưu tiên `replyChatID`, sau đó `chatID`, cuối cùng `senderID`.

## 5. Resolve kỳ hóa đơn

`resolvePeriod(roomID, utilityType, house, forcedPeriod)` xét theo thứ tự, chỉ hỏi user ở bước cuối:

| Thứ tự | Điều kiện | Kỳ được dùng |
|---|---|---|
| 1 | Có `forcedPeriod` | Dùng luôn `forcedPeriod` |
| 2 | Invoice tháng này hoặc tháng trước chưa `PAID` và còn thiếu đúng utility đang nhập | Dùng kỳ của invoice đó |
| 3 | Cả 3 tháng ngay trước tháng hiện tại đều đã có invoice | Dùng tháng hiện tại |
| 4 | Còn lại | Hỏi chọn trong các tháng trước/hiện tại/sau chưa có invoice |

Bước 2 làm cho utility gửi sau (kể cả vài ngày sau) bổ sung vào đúng invoice đang mở, không cần pending
`AWAIT_UTILITY` còn sống. Bước 3 là rule "hóa đơn được tạo đều": lịch sử đều đặn thì kỳ cần ghi chỉ có
thể là tháng hiện tại, nên bot không hỏi lại. `regularInvoiceHistoryMonths = 3` là số tháng liền trước
dùng để kết luận lịch sử đều.

Ví dụ ngày `2026-09-02`:

- Đã có invoice tháng 6, 7, 8 -> `#dien 679qt P201 661` tạo invoice tháng `2026-09`, không hỏi tháng.
- Invoice `2026-09` đã có số điện, chưa có số nước -> `#nuoc 123` cập nhật vào `2026-09`.
- Phòng mới chưa có invoice nào -> hỏi `#8`, `#9` hoặc `#10`.
- Pending lưu `period=2026-05` -> dùng `2026-05`.

`shiftMonth` cộng/trừ tháng từ ngày 1 của tháng, để ngày 29-31 không làm phép trừ tháng nhảy sai tháng.
Trong một lần resolve, `invoiceLookup` cache theo kỳ nên mỗi tháng chỉ query DB một lần.

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

Manager private chat không dùng `#dien <số>` (không xác định được phòng); manager dùng
`#dien <house_code> <room_name> <số>` hoặc batch command.

## 7.1 Flow room target command

Dùng cho `#dien 679qt P201 661` và `#dien P201 661`.

1. Lấy user đã link từ `senderID`.
2. Tenant: chỉ chấp nhận khi tên phòng khớp phòng đang thuê, sai thì nhắc dùng `#dien <số mới>`.
3. Manager: phải là manager của bot này; resolve nhà theo `house_code`, hoặc nhà duy nhất khi bỏ `house_code`.
4. Manager nhiều nhà mà bỏ `house_code` -> báo lỗi kèm hướng dẫn.
5. Match tên phòng trong nhà rồi chạy tiếp flow single command.

Lệnh này ghi đè pending `AWAIT_UTILITY` của chat (xóa pending rồi xử lý như lệnh mới), vì nó có thể trỏ
sang phòng khác và bước 2 của `resolvePeriod` đã tự tìm được invoice đang mở.

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

User: #dien 679qt P201 500
Bot: Vui lòng bổ sung số nước tháng 05/2026 bằng cú pháp: #nuoc 679qt p201 <số mới> hoặc #huy để hủy.
```

`followUpRoomTarget` quyết định phần `<house_code> <room_name>` trong câu nhắc: chat đã xác định được
phòng thì nhắc cú pháp ngắn, lệnh có nêu phòng thì nhắc lại đúng phòng đó.

Pending lưu tối thiểu:

```json
{"utility_type":"nuoc","room_id":"room-id","period":"2026-05"}
```

Rule quan trọng:

- User gửi đúng utility còn thiếu -> dùng `period` trong pending làm `forcedPeriod`.
- Không resolve kỳ mới trong `AWAIT_UTILITY`.
- User gửi sai utility -> nhắc utility đang chờ, giữ pending.
- User gửi thiếu chỉ số -> báo lỗi, giữ pending để retry.
- User gửi lệnh có nêu phòng -> xóa pending, xử lý như lệnh mới.
- Pending mất (bị lệnh khác thay thế) vẫn bổ sung đúng invoice nhờ bước 2 của `resolvePeriod`.

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

## 13. Lệnh manager sửa dữ liệu

Cả hai lệnh đều đi qua `resolveManagerCommandRoom`:

1. `ensureManagerSender` — chỉ quản lý của bot này dùng được.
2. Có room target -> resolve nhà (`house_code` hoặc nhà duy nhất) rồi `MatchRoom`.
3. Không có room target -> lấy phòng đang kết nối với group chat hiện tại, và kiểm tra nhà thuộc manager.
4. Không có room target trong chat riêng -> báo lỗi kèm ví dụ cú pháp.
5. Không có room target trong group chưa kết nối -> báo lỗi kèm luôn Group ID và cú pháp `#update-room`.

### 13.1 `#update-tenant`

- Lấy danh sách khách thuê active của phòng bằng `ListTenantByRoomID`.
- 0 khách thuê -> báo phòng chưa có khách thuê.
- Nhiều hơn 1 khách thuê -> liệt kê tên + số hiện tại và yêu cầu cập nhật trên web, **không đoán**.
  Lệnh chat không có cách nào chỉ định một khách thuê cụ thể.
- Đúng 1 khách thuê -> gọi `TenantService.UpdateTenantInfo` với `Phone`, để dùng lại rule trùng số
  điện thoại (`model.ErrPhoneAlreadyExists`) của service thay vì viết lại.
- Số mới trùng số cũ -> chỉ báo lại, không ghi.
- Khách thuê chưa link Zalo -> nhắc manager bảo họ gửi "bot ơi" + số mới để link.

### 13.2 `#update-room`

- Xác định `group_chat_id`: token cuối, hoặc chat ID hiện tại khi gửi trong group.
- Trùng với `group_chat_id` đang lưu -> chỉ báo lại.
- Group ID đã thuộc phòng khác -> báo tên phòng đó, yêu cầu xóa kết nối cũ trên web.
  Một group chat chỉ map tới một phòng, vì `GetRoomByGroupChatID` phải trả về duy nhất.
- Ghi bằng `linkRoomToGroupChat`: copy toàn bộ field của phòng rồi đổi `GroupChatID`, vì
  `RoomRepository.UpdateRoom` ghi đè cả dòng. `autoLinkRoom` dùng chung helper này.

Message "nhóm chưa kết nối" của `handleGroupLinking` bị bỏ qua khi text là `#help`,
`#update-room` hoặc `#update-tenant` (`groupLinkingReplySuppressed`), vì các handler đó đã tự trả lời
kèm Group ID.

## 14. Lệnh #help

`#help` chạy trước mọi flow linking và invoice, nên luôn trả lời được. Nội dung phụ thuộc vai trò và
trạng thái liên kết của người gửi:

| Ngữ cảnh | Nội dung trả lời |
|---|---|
| Group chưa kết nối phòng | Hướng dẫn kết nối: copy Group ID, dán vào "Group Chat ID" của phòng trên web |
| Group đã kết nối phòng | `#dien/#nuoc <số mới>`, `#ok`, `#huy`, nhắc phải @ bot trong nhóm |
| Private, sender chưa link, manager chưa link Zalo | Hướng dẫn manager nhắn mật khẩu đăng nhập để kích hoạt |
| Private, sender chưa link, manager đã link Zalo | Hướng dẫn tenant nhắn "bot ơi" rồi gửi số điện thoại |
| Private, manager của bot | Toàn bộ cú pháp manager (gồm `#update-tenant`, `#update-room`), kèm cú pháp rút gọn nếu chỉ có 1 nhà |
| Private, tenant của manager | `#dien/#nuoc <số mới>`, `#ok`, `#huy`, nhắc gửi ảnh chuyển khoản vào nhóm |

Rule: người chưa liên kết chỉ nhận hướng dẫn liên kết, không nhận danh sách lệnh, vì chưa lệnh nào chạy
được cho họ. Khi group chưa kết nối, `handleGroupLinking` nhường lượt trả lời cho `#help` để không gửi
hai tin trùng nội dung.

## 15. Rule maintain cần giữ

- Reply command phải có `#`.
- `AWAIT_UTILITY` luôn dùng `period` trong pending.
- Không xóa `AWAIT_UTILITY` khi user nhập sai utility hoặc thiếu chỉ số.
- Không bỏ chữ cái khi normalize phòng.
- Không đổi logic zero usage nếu nghiệp vụ vẫn yêu cầu user nhập lại số cũ.
- Thêm pending action mới thì cập nhật `handlePendingCommand` và test tương ứng.
- `#help` phải chạy trước flow linking, nếu không manager chưa link sẽ chỉ nhận prompt nhập mật khẩu.
- Thêm command mới thì cập nhật cả `#help` và bảng cú pháp ở mục 2.
- `#update-tenant` không được tự chọn khách thuê khi phòng có nhiều người.
- `linkRoomToGroupChat` phải copy đủ field của phòng, nếu thiếu sẽ xóa giá/phụ phí riêng của phòng.
