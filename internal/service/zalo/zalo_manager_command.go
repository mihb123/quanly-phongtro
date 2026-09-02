package zalo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"
)

// handleUpdateTenantPhoneCommand applies "#update-tenant [<mã nhà>] [<phòng>] <số điện thoại>",
// replacing whatever phone the room's tenant has today.
func (s *zaloInvoiceCommandServiceImpl) handleUpdateTenantPhoneCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand) error {
	chatID := commandChatID(webhookCtx)
	if parsed.Value == "" {
		return s.sendTextMessage(ctx, managerID, chatID, "Số điện thoại không hợp lệ. Cú pháp: #update-tenant <mã nhà> <phòng> <số điện thoại>\nVí dụ: #update-tenant 679qt P201 0912345678")
	}
	if s.tenantService == nil {
		return s.sendTextMessage(ctx, managerID, chatID, "Chức năng cập nhật khách thuê chưa được bật.")
	}

	room, err := s.resolveManagerCommandRoom(ctx, managerID, webhookCtx, parsed, "#update-tenant")
	if err != nil {
		return s.sendTextMessage(ctx, managerID, chatID, err.Error())
	}

	// --- 1. PICK THE TENANT ---
	// The command has no way to name one tenant, so a shared room is reported instead of guessed.
	tenants, err := s.tenantRepo.ListTenantByRoomID(ctx, managerID, room.ID)
	if err != nil {
		return err
	}
	if len(tenants) == 0 {
		return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("Phòng %s chưa có khách thuê nào.", room.Name))
	}
	if len(tenants) > 1 {
		return s.sendTextMessage(ctx, managerID, chatID, manyTenantsMessage(room.Name, tenants))
	}

	// --- 2. UPDATE THE PHONE ---
	tenant := tenants[0]
	if tenant.Phone == parsed.Value {
		return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("%s (phòng %s) đang dùng đúng số %s rồi.", tenant.FullName, room.Name, parsed.Value))
	}

	phone := parsed.Value
	if _, err := s.tenantService.UpdateTenantInfo(ctx, managerID, tenant.TenantID, tenantsvc.UpdateTenantInput{Phone: &phone}); err != nil {
		if errors.Is(err, model.ErrPhoneAlreadyExists) {
			return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("Số %s đã được dùng cho một tài khoản khác. Vui lòng kiểm tra lại.", phone))
		}
		return s.sendTextMessage(ctx, managerID, chatID, "Không cập nhật được số điện thoại: "+err.Error())
	}

	previous := tenant.Phone
	if previous == "" {
		previous = "chưa có"
	}
	message := fmt.Sprintf("✅ Đã cập nhật số điện thoại của %s (phòng %s): %s → %s.", tenant.FullName, room.Name, previous, phone)
	if tenant.ZaloUserID == "" {
		message += fmt.Sprintf("\nKhách thuê chưa liên kết Zalo. Nhắn họ gửi \"bot ơi\" rồi gửi số %s cho bot.", phone)
	}
	return s.sendTextMessage(ctx, managerID, chatID, message)
}

// handleUpdateRoomGroupCommand applies "#update-room [<mã nhà>] [<phòng>] [<mã nhóm>]". Sent inside
// a room's Zalo group without a group ID it connects that very group, which is the whole point: the
// manager no longer has to copy the ID over to the web.
func (s *zaloInvoiceCommandServiceImpl) handleUpdateRoomGroupCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand) error {
	chatID := commandChatID(webhookCtx)

	groupChatID := parsed.Value
	if groupChatID == "" {
		if !webhookCtx.isGroupChat {
			return s.sendTextMessage(ctx, managerID, chatID, "Vui lòng nhập mã nhóm. Cú pháp: #update-room <mã nhà> <phòng> <mã nhóm>\nHoặc gửi #update-room <mã nhà> <phòng> ngay trong nhóm Zalo của phòng để kết nối nhóm đó.")
		}
		groupChatID = webhookCtx.chatID
	}

	room, err := s.resolveManagerCommandRoom(ctx, managerID, webhookCtx, parsed, "#update-room")
	if err != nil {
		return s.sendTextMessage(ctx, managerID, chatID, err.Error())
	}

	if room.GroupChatID != nil && *room.GroupChatID == groupChatID {
		return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("Phòng %s đã được kết nối với nhóm %s.", room.Name, groupChatID))
	}
	// One group chat maps to one room, so an ID already in use has to be freed first.
	if existing, err := s.roomRepo.GetRoomByGroupChatID(ctx, groupChatID); err == nil && existing != nil && existing.ID != room.ID {
		return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("Mã nhóm %s đang được kết nối với phòng %s. Vui lòng xóa kết nối cũ trên web trước.", groupChatID, existing.Name))
	}

	if err := linkRoomToGroupChat(ctx, s.roomRepo, *room, room.HouseID, groupChatID); err != nil {
		return s.sendTextMessage(ctx, managerID, chatID, "Không cập nhật được mã nhóm: "+err.Error())
	}

	if room.GroupChatID != nil && *room.GroupChatID != "" {
		return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("✅ Đã đổi nhóm Zalo của phòng %s: %s → %s.", room.Name, *room.GroupChatID, groupChatID))
	}
	return s.sendTextMessage(ctx, managerID, chatID, fmt.Sprintf("✅ Đã kết nối phòng %s với nhóm Zalo %s.", room.Name, groupChatID))
}

// resolveManagerCommandRoom resolves the room a manager update command targets: the room named in
// the command, or the room already linked to the group chat the command was sent from.
func (s *zaloInvoiceCommandServiceImpl) resolveManagerCommandRoom(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, commandName string) (*model.Room, error) {
	if err := s.ensureManagerSender(ctx, managerID, webhookCtx.senderID); err != nil {
		return nil, err
	}

	if parsed.RoomName == "" {
		if !webhookCtx.isGroupChat {
			return nil, fmt.Errorf("Vui lòng nhập phòng cần cập nhật. Ví dụ: %s 679qt P201 <giá trị mới>", commandName)
		}
		room, err := s.roomRepo.GetRoomByGroupChatID(ctx, webhookCtx.chatID)
		if err != nil {
			// Carry the group ID so this single reply replaces the "group not connected" notice.
			return nil, fmt.Errorf("Nhóm này chưa được kết nối với phòng nào.\nMã nhóm (Group ID): %s\nVui lòng nhắn: %s <mã nhà> <phòng>", webhookCtx.chatID, commandName)
		}
		if _, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID); err != nil {
			return nil, errors.New("Group chat này không thuộc quản lý hiện tại.")
		}
		return room, nil
	}

	house, roomName, err := s.resolveCommandHouse(ctx, managerID, parsed.HouseCode, parsed.RoomName)
	if err != nil {
		return nil, err
	}
	rooms, err := s.roomRepo.ListAllRoomsByHouseID(ctx, house.ID)
	if err != nil {
		return nil, err
	}
	room, err := MatchRoom(roomName, rooms)
	if err != nil {
		if errors.Is(err, ErrAmbiguousRoomName) {
			return nil, fmt.Errorf("Nhà %s có nhiều phòng trùng tên %s. Vui lòng nhập tên phòng đầy đủ.", house.Name, roomName)
		}
		return nil, fmt.Errorf("Không tìm thấy phòng %s trong nhà %s.", roomName, house.Name)
	}
	return room, nil
}

// linkRoomToGroupChat stores a Zalo group chat ID on a room. Every other field is copied from the
// current room because the repository update replaces the whole row.
func linkRoomToGroupChat(ctx context.Context, roomRepo model.RoomRepository, room model.Room, houseID, groupChatID string) error {
	params := model.UpdateRoomParams{
		Name:                  room.Name,
		Price:                 room.Price,
		MaxTenants:            room.MaxTenants,
		Status:                room.Status,
		ElectricityPrice:      room.ElectricityPrice,
		WaterPrice:            room.WaterPrice,
		WifiPrice:             room.WifiPrice,
		ParkingPrice:          room.ParkingPrice,
		ServicePrice:          room.ServicePrice,
		ExtraPersonThreshold:  room.ExtraPersonThreshold,
		ExtraPersonFee:        room.ExtraPersonFee,
		ExtraVehicleThreshold: room.ExtraVehicleThreshold,
		ExtraVehicleFee:       room.ExtraVehicleFee,
		GroupChatID:           &groupChatID,
	}
	_, err := roomRepo.UpdateRoom(ctx, room.ID, houseID, params)
	return err
}

// manyTenantsMessage lists the tenants of a shared room, which a chat command cannot disambiguate.
func manyTenantsMessage(roomName string, tenants []model.FullInfoTenant) string {
	lines := []string{fmt.Sprintf("Phòng %s có %d khách thuê nên bot không biết cập nhật cho ai:", roomName, len(tenants))}
	for _, tenant := range tenants {
		phone := tenant.Phone
		if phone == "" {
			phone = "chưa có số"
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", tenant.FullName, phone))
	}
	return strings.Join(append(lines, "Vui lòng cập nhật trên web để chọn đúng khách thuê."), "\n")
}

// groupLinkingReplySuppressed reports whether a group message already gets a better answer from its
// own handler, so the "group not connected" notice would only duplicate it.
func groupLinkingReplySuppressed(text string) bool {
	switch ParseCommand(text).Type {
	case CommandHelp, CommandUpdateRoomGroup, CommandUpdateTenantPhone:
		return true
	default:
		return false
	}
}
