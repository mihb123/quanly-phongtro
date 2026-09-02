package zalo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

// handleSingleCommand resolves a group or tenant private command to one room.
func (s *zaloInvoiceCommandServiceImpl) handleSingleCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, forcedPeriod string, allowOverwrite bool) error {
	room, err := s.resolveSingleCommandRoom(ctx, managerID, webhookCtx)
	if err != nil {
		_ = s.sendTextMessage(ctx, managerID, commandChatID(webhookCtx), err.Error())
		return nil
	}
	return s.processSingleRoom(ctx, managerID, webhookCtx, parsed, *room, forcedPeriod, allowOverwrite)
}

// processSingleRoom applies a utility reading to one resolved room.
func (s *zaloInvoiceCommandServiceImpl) processSingleRoom(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, room model.Room, forcedPeriod string, allowOverwrite bool) error {
	entry := RoomUtilityEntry{}
	if len(parsed.Entries) > 0 {
		entry = parsed.Entries[0]
	}

	result, pendingCreated, err := s.applyUtilityUpdate(ctx, managerID, webhookCtx, utilityUpdateRequest{
		UtilityType:    parsed.UtilityType,
		Room:           room,
		NewIndex:       entry.NewIndex,
		HasNewIndex:    entry.HasNewIndex,
		ForcedPeriod:   forcedPeriod,
		AllowOverwrite: allowOverwrite,
	})
	if pendingCreated {
		return err
	}
	if err != nil {
		return s.sendTextMessage(ctx, managerID, commandChatID(webhookCtx), err.Error())
	}
	return s.respondAfterSingleUpdate(ctx, managerID, commandChatID(webhookCtx), result, followUpRoomTarget(parsed))
}

// followUpRoomTarget returns the "<mã nhà> <phòng>" part a follow-up command must repeat. It is
// empty for chats that already identify the room, where the bare "#nuoc <số>" form is enough.
func followUpRoomTarget(parsed *ParsedCommand) string {
	if parsed.Type != CommandUtilityRoom || len(parsed.Entries) == 0 {
		return ""
	}
	if parsed.HouseCode == "" {
		return parsed.Entries[0].RoomName
	}
	return parsed.HouseCode + " " + parsed.Entries[0].RoomName
}

// handleRoomTargetCommand applies a reading to the room named inside the command text, e.g.
// "#dien 679qt P201 661" or the short "#dien P201 661" when the sender manages a single house.
func (s *zaloInvoiceCommandServiceImpl) handleRoomTargetCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, forcedPeriod string, allowOverwrite bool) error {
	room, err := s.resolveRoomTargetRoom(ctx, managerID, webhookCtx, parsed)
	if err != nil {
		_ = s.sendTextMessage(ctx, managerID, commandChatID(webhookCtx), err.Error())
		return nil
	}
	return s.processSingleRoom(ctx, managerID, webhookCtx, parsed, *room, forcedPeriod, allowOverwrite)
}

// resolveRoomTargetRoom finds the room a named command addresses. Managers may address any room in
// their houses; a tenant may only name their own room, so the name acts as a confirmation there.
func (s *zaloInvoiceCommandServiceImpl) resolveRoomTargetRoom(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand) (*model.Room, error) {
	roomName := ""
	if len(parsed.Entries) > 0 {
		roomName = parsed.Entries[0].RoomName
	}
	if roomName == "" {
		return nil, errors.New("Vui lòng nhập tên phòng. Ví dụ: #dien 679qt P201 661")
	}

	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
	if err != nil {
		return nil, errors.New("Tài khoản Zalo này chưa liên kết với hệ thống.")
	}

	if linkedUser.Role != model.RoleManager {
		return s.tenantOwnRoom(ctx, managerID, linkedUser.ID, roomName)
	}
	if linkedUser.ID != managerID {
		return nil, errors.New("Chỉ quản lý của bot này mới được cập nhật theo tên phòng.")
	}

	house, roomName, err := s.resolveCommandHouse(ctx, managerID, parsed.HouseCode, roomName)
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

// tenantOwnRoom returns the tenant's room only when the named room is the one they rent.
func (s *zaloInvoiceCommandServiceImpl) tenantOwnRoom(ctx context.Context, managerID, userID, roomName string) (*model.Room, error) {
	tenant, err := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, userID)
	if err != nil {
		return nil, errors.New("Không tìm thấy phòng đang thuê cho tài khoản này.")
	}
	room, err := s.roomRepo.GetRoomByIDOnly(ctx, tenant.RoomID)
	if err != nil {
		return nil, err
	}
	if NormalizeRoomName(room.Name) != NormalizeRoomName(roomName) {
		return nil, fmt.Errorf("Bạn chỉ có thể cập nhật cho phòng %s của mình. Chỉ cần nhắn: #dien <số mới>", room.Name)
	}
	return room, nil
}

// resolveCommandHouse picks the house of a named room command. The house code may be omitted when
// the manager has exactly one house, and a code that matches no house is retried as the first word
// of a multi-word room name ("#dien Phòng 201 661").
func (s *zaloInvoiceCommandServiceImpl) resolveCommandHouse(ctx context.Context, managerID, houseCode, roomName string) (*model.House, string, error) {
	singleHouse, err := s.singleManagedHouse(ctx, managerID)
	if err != nil {
		return nil, "", err
	}

	if houseCode == "" {
		if singleHouse == nil {
			return nil, "", errors.New("Bạn đang quản lý nhiều nhà. Vui lòng nhập mã nhà. Ví dụ: #dien 679qt P201 661")
		}
		return singleHouse, roomName, nil
	}

	house, err := s.houseRepo.GetHouseByCode(ctx, managerID, houseCode)
	if err == nil {
		return house, roomName, nil
	}
	if singleHouse == nil {
		return nil, "", fmt.Errorf("Không tìm thấy nhà với mã: %s", houseCode)
	}
	return singleHouse, strings.TrimSpace(houseCode + " " + roomName), nil
}

// singleManagedHouse returns the manager's only house, or nil when they manage several.
func (s *zaloInvoiceCommandServiceImpl) singleManagedHouse(ctx context.Context, managerID string) (*model.House, error) {
	houses, err := s.houseRepo.ListHouseByManagerID(ctx, managerID, 2, 0, "")
	if err != nil {
		return nil, err
	}
	if len(houses) != 1 {
		return nil, nil
	}
	house := houses[0]
	return &house, nil
}

// processAwaitUtilitySingleCommand applies a missing utility without losing the saved period on retryable errors.
func (s *zaloInvoiceCommandServiceImpl) processAwaitUtilitySingleCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, pendingID string, room model.Room, forcedPeriod string) error {
	entry := RoomUtilityEntry{}
	if len(parsed.Entries) > 0 {
		entry = parsed.Entries[0]
	}

	result, pendingCreated, err := s.applyUtilityUpdate(ctx, managerID, webhookCtx, utilityUpdateRequest{
		UtilityType:  parsed.UtilityType,
		Room:         room,
		NewIndex:     entry.NewIndex,
		HasNewIndex:  entry.HasNewIndex,
		ForcedPeriod: forcedPeriod,
	})
	if pendingCreated {
		return err
	}
	if err != nil {
		return s.sendTextMessage(ctx, managerID, commandChatID(webhookCtx), err.Error())
	}
	if result.Complete {
		if err := s.pendingRepo.DeleteByID(ctx, pendingID); err != nil {
			return err
		}
	}
	return s.respondAfterSingleUpdate(ctx, managerID, commandChatID(webhookCtx), result, "")
}

// handleBatchCommand processes manager private commands for many rooms in one house.
func (s *zaloInvoiceCommandServiceImpl) handleBatchCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, forcedPeriod string, allowOverwrite bool) error {
	chatID := commandChatID(webhookCtx)
	if webhookCtx.isGroupChat {
		return s.sendTextMessage(ctx, managerID, chatID, "Lệnh nhiều phòng chỉ dùng trong chat riêng với quản lý.")
	}
	if err := s.ensureManagerSender(ctx, managerID, webhookCtx.senderID); err != nil {
		return s.sendTextMessage(ctx, managerID, chatID, err.Error())
	}
	if parsed.HouseCode == "" {
		return s.sendTextMessage(ctx, managerID, chatID, "Vui lòng nhập mã nhà. Ví dụ: #dien 679qt")
	}
	if len(parsed.Entries) == 0 {
		return s.sendTextMessage(ctx, managerID, chatID, "Chưa có dòng phòng hợp lệ. Ví dụ: P101 750")
	}

	house, err := s.houseRepo.GetHouseByCode(ctx, managerID, parsed.HouseCode)
	if err != nil {
		return s.sendTextMessage(ctx, managerID, chatID, "Không tìm thấy nhà với mã: "+parsed.HouseCode)
	}
	rooms, err := s.roomRepo.ListAllRoomsByHouseID(ctx, house.ID)
	if err != nil {
		return err
	}

	var matchedEntries []matchedBatchEntry
	var failed []string
	for _, entry := range parsed.Entries {
		room, err := MatchRoom(entry.RoomName, rooms)
		if err != nil {
			failed = append(failed, fmt.Sprintf("- %s: không tìm thấy phòng", entry.RoomName))
			continue
		}
		matchedEntries = append(matchedEntries, matchedBatchEntry{Entry: entry, Room: *room})
	}

	for _, matchedEntry := range matchedEntries {
		resolution, err := s.resolvePeriod(ctx, matchedEntry.Room.ID, parsed.UtilityType, house, forcedPeriod)
		if err != nil {
			failed = append(failed, fmt.Sprintf("- %s: %v", matchedEntry.Room.Name, err))
			continue
		}
		if resolution.NeedsSelection {
			if err := s.createBatchPeriodPending(ctx, managerID, webhookCtx, parsed, resolution.Options); err != nil {
				return err
			}
			return s.sendTextMessage(ctx, managerID, chatID, periodSelectionMessage(house.Name, resolution.Options))
		}

		existingInvoice, err := s.invoiceRepo.GetInvoiceByRoomAndPeriod(ctx, matchedEntry.Room.ID, resolution.Period)
		if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
			failed = append(failed, fmt.Sprintf("- %s: %v", matchedEntry.Room.Name, err))
			continue
		}
		if existingInvoice != nil && shouldConfirmOverwrite(existingInvoice, house, utilityUpdateRequest{
			UtilityType:    parsed.UtilityType,
			Room:           matchedEntry.Room,
			NewIndex:       matchedEntry.Entry.NewIndex,
			HasNewIndex:    matchedEntry.Entry.HasNewIndex,
			ForcedPeriod:   forcedPeriod,
			AllowOverwrite: allowOverwrite,
		}) && !allowOverwrite {
			if err := s.createBatchOverwritePending(ctx, managerID, webhookCtx, parsed); err != nil {
				return err
			}
			message := fmt.Sprintf("Một số phòng trong nhà %s đã có số %s cho kỳ cần cập nhật.\nNhắn '#ok' để xác nhận ghi đè, '#huy' để hủy.", house.Name, utilityLabel(parsed.UtilityType))
			return s.sendTextMessage(ctx, managerID, chatID, message)
		}
	}

	var results []utilityUpdateResult
	for _, matchedEntry := range matchedEntries {
		entry := matchedEntry.Entry

		result, pendingCreated, err := s.applyUtilityUpdate(ctx, managerID, webhookCtx, utilityUpdateRequest{
			UtilityType:    parsed.UtilityType,
			Room:           matchedEntry.Room,
			NewIndex:       entry.NewIndex,
			HasNewIndex:    entry.HasNewIndex,
			ForcedPeriod:   forcedPeriod,
			AllowOverwrite: allowOverwrite,
			IsBatch:        true,
		})
		if pendingCreated {
			return nil
		}
		if err != nil {
			failed = append(failed, fmt.Sprintf("- %s: %v", entry.RoomName, err))
			continue
		}
		results = append(results, result)
	}

	return s.respondAfterBatchUpdate(ctx, managerID, chatID, *house, parsed.UtilityType, results, failed)
}

// resolveSingleCommandRoom maps a Zalo group or tenant private chat to one room.
func (s *zaloInvoiceCommandServiceImpl) resolveSingleCommandRoom(ctx context.Context, managerID string, webhookCtx webhookMessageContext) (*model.Room, error) {
	if webhookCtx.isGroupChat {
		room, err := s.roomRepo.GetRoomByGroupChatID(ctx, webhookCtx.chatID)
		if err != nil {
			return nil, errors.New("Không tìm thấy phòng liên kết với group chat này.")
		}
		if _, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID); err != nil {
			return nil, errors.New("Group chat này không thuộc quản lý hiện tại.")
		}
		return room, nil
	}

	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
	if err != nil {
		return nil, errors.New("Tài khoản Zalo này chưa liên kết với hệ thống.")
	}
	if linkedUser.Role == model.RoleManager {
		return nil, errors.New("Quản lý vui lòng nhập kèm phòng: #dien <mã nhà> <phòng> <số mới>. Nhắn #help để xem tất cả cú pháp.")
	}

	tenant, err := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, linkedUser.ID)
	if err != nil {
		return nil, errors.New("Không tìm thấy phòng đang thuê cho tài khoản này.")
	}
	return s.roomRepo.GetRoomByIDOnly(ctx, tenant.RoomID)
}

// ensureManagerSender verifies that a manager-only command came from the linked manager.
func (s *zaloInvoiceCommandServiceImpl) ensureManagerSender(ctx context.Context, managerID, senderID string) error {
	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, senderID)
	if err != nil {
		return errors.New("Tài khoản Zalo này chưa liên kết với hệ thống.")
	}
	if linkedUser.ID != managerID || linkedUser.Role != model.RoleManager {
		return errors.New("Chỉ quản lý mới được dùng lệnh này.")
	}
	return nil
}
