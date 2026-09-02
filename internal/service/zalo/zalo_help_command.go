package zalo

import (
	"context"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

// isHelpCommandText reports whether a message asks for the command list.
func isHelpCommandText(text string) bool {
	return ParseCommand(text).Type == CommandHelp
}

// handleHelpCommand answers "#help" with the commands the sender can actually use. It runs before
// any account-linking or invoice logic so an unlinked user still gets an answer.
func (s *zaloServiceImpl) handleHelpCommand(ctx context.Context, managerID string, manager *model.User, webhookCtx webhookMessageContext) {
	chatID := commandChatID(webhookCtx)
	if chatID == "" {
		return
	}

	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		logger.Error(nil, 0, "zalo help get token failed", err)
		return
	}
	if err := s.client.SendMessage(ctx, botToken, chatID, s.buildHelpMessage(ctx, managerID, manager, webhookCtx)); err != nil {
		logger.Error(nil, 0, "zalo help send message failed", err)
	}
}

// buildHelpMessage picks the help text matching the sender's role and connection state. Anyone who
// is not connected yet only gets the connection instructions, because no command works for them.
func (s *zaloServiceImpl) buildHelpMessage(ctx context.Context, managerID string, manager *model.User, webhookCtx webhookMessageContext) string {
	managerLinked := manager.ZaloUserID != nil && *manager.ZaloUserID != ""

	if webhookCtx.isGroupChat {
		room, err := s.roomRepo.GetRoomByGroupChatID(ctx, webhookCtx.chatID)
		if err != nil || room == nil {
			return helpGroupNotConnectedMessage(webhookCtx.chatID)
		}
		return helpGroupMessage(room.Name)
	}

	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
	if err != nil || linkedUser == nil {
		return helpNotConnectedMessage(managerLinked)
	}

	if linkedUser.Role == model.RoleManager {
		if linkedUser.ID != managerID {
			return helpNotConnectedMessage(managerLinked)
		}
		return helpManagerMessage(s.managerHouseCodeHint(ctx, managerID))
	}

	tenant, err := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, linkedUser.ID)
	if err != nil || tenant == nil {
		return helpNotConnectedMessage(managerLinked)
	}
	return helpTenantMessage(tenant.RoomName)
}

// managerHouseCodeHint returns the manager's only house code so the help can show the short syntax
// that omits the house, and an empty string when they manage several houses.
func (s *zaloServiceImpl) managerHouseCodeHint(ctx context.Context, managerID string) string {
	houses, err := s.houseRepo.ListHouseByManagerID(ctx, managerID, 2, 0, "")
	if err != nil || len(houses) != 1 {
		return ""
	}
	return houses[0].HouseCode
}

// helpNotConnectedMessage explains how to link a Zalo account. Before the manager links their own
// account they are the only person who can act, so their step is the only one worth showing.
func helpNotConnectedMessage(managerLinked bool) string {
	if !managerLinked {
		return strings.Join([]string{
			"👋 Zalo của bạn chưa được liên kết với hệ thống quản lý trọ.",
			"",
			"▪️ Nếu bạn là QUẢN LÝ (chủ bot này):",
			"Nhắn mật khẩu đăng nhập của bạn vào đây để kích hoạt bot.",
			"",
			"▪️ Nếu bạn là KHÁCH THUÊ:",
			"Vui lòng chờ quản lý liên kết Zalo trước, sau đó nhắn \"bot ơi\" để bắt đầu.",
			"",
			"Liên kết xong, nhắn #help để xem danh sách lệnh.",
		}, "\n")
	}

	return strings.Join([]string{
		"👋 Zalo của bạn chưa được liên kết với hệ thống quản lý trọ.",
		"",
		"Cách liên kết:",
		"1. Nhắn \"bot ơi\" cho bot.",
		"2. Gửi số điện thoại bạn đã đăng ký với quản lý (VD: 0912345678).",
		"",
		"Nếu bot báo số điện thoại không đúng, vui lòng liên hệ quản lý để kiểm tra lại thông tin.",
		"Liên kết xong, nhắn #help để xem danh sách lệnh.",
	}, "\n")
}

// helpManagerMessage lists the manager commands, including the short form when houseCode is set.
func helpManagerMessage(houseCode string) string {
	lines := []string{
		"📋 DANH SÁCH LỆNH — QUẢN LÝ",
		"",
		"1. Ghi số điện/nước cho 1 phòng",
		"#dien <mã nhà> <phòng> <số mới>",
		"#nuoc <mã nhà> <phòng> <số mới>",
		"VD: #dien 679qt P201 661",
	}
	if houseCode != "" {
		lines = append(lines, fmt.Sprintf("Bạn chỉ quản lý 1 nhà (%s) nên có thể bỏ mã nhà: #dien P201 661", houseCode))
	}

	lines = append(lines,
		"",
		"2. Ghi số cho nhiều phòng (mỗi phòng 1 dòng)",
		"#dien <mã nhà>",
		"P101 750",
		"P201 900",
		"",
		"3. Bổ sung số còn thiếu",
		"Hóa đơn tính cả điện và nước: gõ 1 loại trước, loại còn lại gõ sau (kể cả vài ngày sau), hệ thống tự bổ sung vào đúng hóa đơn đang mở.",
		"VD: #dien 679qt P201 661, hôm sau #nuoc 679qt P201 123",
		"",
		"4. Đổi số điện thoại khách thuê",
		"#update-tenant <mã nhà> <phòng> <số điện thoại>",
		"VD: #update-tenant 679qt P201 0912345678",
		"Có số cũ thì ghi đè bằng số mới. Phòng nhiều khách thuê thì cập nhật trên web.",
		"",
		"5. Kết nối nhóm Zalo với phòng",
		"Nhắn ngay trong nhóm của phòng: #update-room <mã nhà> <phòng>",
		"VD: #update-room 679qt P201",
		"Hoặc nhập mã nhóm từ chat riêng: #update-room 679qt P201 <mã nhóm>",
		"",
		"6. Trả lời bot",
		"#ok — xác nhận",
		"#huy — hủy thao tác đang chờ",
		"#<số tháng> — chọn tháng khi bot hỏi (VD: #9)",
		"",
		"7. #help — xem lại danh sách lệnh",
		"",
		"Trong nhóm Zalo của phòng, hãy @ bot (hoặc reply tin nhắn của bot) để bot nhận được lệnh.",
	)
	return strings.Join(lines, "\n")
}

// helpTenantMessage lists the short commands available to a tenant whose room is already known.
func helpTenantMessage(roomName string) string {
	title := "📋 DANH SÁCH LỆNH — KHÁCH THUÊ"
	if roomName != "" {
		title = fmt.Sprintf("📋 DANH SÁCH LỆNH — KHÁCH THUÊ (%s)", roomName)
	}

	return strings.Join([]string{
		title,
		"",
		"#dien <số mới> — gửi số điện mới. VD: #dien 661",
		"#nuoc <số mới> — gửi số nước mới. VD: #nuoc 123",
		"Bot đã biết bạn ở phòng nào nên không cần nhập tên nhà hay tên phòng.",
		"",
		"#ok — xác nhận | #huy — hủy thao tác đang chờ",
		"#help — xem lại danh sách lệnh",
		"",
		"Gửi ảnh chuyển khoản vào nhóm Zalo của phòng để hệ thống ghi nhận thanh toán.",
		"Trong nhóm, hãy @ bot để bot nhận được lệnh.",
	}, "\n")
}

// helpGroupMessage lists the commands usable inside a group already connected to a room.
func helpGroupMessage(roomName string) string {
	return strings.Join([]string{
		fmt.Sprintf("📋 DANH SÁCH LỆNH — NHÓM %s", roomName),
		"",
		"#dien <số mới> — gửi số điện mới. VD: #dien 661",
		"#nuoc <số mới> — gửi số nước mới. VD: #nuoc 123",
		"#ok — xác nhận | #huy — hủy thao tác đang chờ",
		"#help — xem lại danh sách lệnh",
		"",
		"Lưu ý: trong nhóm, hãy @ bot (hoặc reply tin nhắn của bot) để bot nhận được lệnh.",
		"Gửi ảnh chuyển khoản vào nhóm để hệ thống ghi nhận thanh toán hóa đơn.",
	}, "\n")
}

// helpGroupNotConnectedMessage tells the manager how to connect an unlinked group to a room.
func helpGroupNotConnectedMessage(groupChatID string) string {
	return strings.Join([]string{
		"⚠️ Nhóm này chưa được kết nối với phòng nào nên bot chưa nhận lệnh ở đây.",
		"",
		"Cách nhanh nhất (quản lý nhắn ngay trong nhóm này):",
		"#update-room <mã nhà> <phòng>",
		"VD: #update-room 679qt P201",
		"",
		"Hoặc kết nối trên web:",
		"1. Sao chép mã nhóm (Group ID) sau:",
		groupChatID,
		"2. Trên web, mở phòng cần kết nối → Sửa phòng → dán mã vào ô \"Group Chat ID\" → Lưu.",
		"",
		"Kết nối xong, nhắn #help trong nhóm để xem danh sách lệnh.",
	}, "\n")
}
