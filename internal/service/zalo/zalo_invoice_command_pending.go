package zalo

import (
	"context"
	"strconv"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type ZaloInvoiceCommandService interface {
	HandleInvoiceCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext) error
	HasPendingState(ctx context.Context, managerID, chatID string) bool
}

// HasPendingState reports whether a chat has a blocking pending command state.
func (s *zaloInvoiceCommandServiceImpl) HasPendingState(ctx context.Context, managerID, chatID string) bool {
	if s.pendingRepo == nil || chatID == "" {
		return false
	}
	pending, err := s.pendingRepo.GetByChatID(ctx, managerID, chatID)
	if err != nil {
		return false
	}
	return pending.ActionType == model.PendingActionConfirmOverwrite || pending.ActionType == model.PendingActionAwaitPeriod
}

// handlePendingCommand applies a chat reply to a saved confirmation or period selection.
func (s *zaloInvoiceCommandServiceImpl) handlePendingCommand(ctx context.Context, managerID, chatID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, pending *model.PendingInvoiceUpdate) (bool, error) {
	if parsed.Type == CommandCancel {
		if err := s.pendingRepo.DeleteByID(ctx, pending.ID); err != nil {
			return true, err
		}
		return true, s.sendTextMessage(ctx, managerID, chatID, "Đã hủy thao tác hóa đơn đang chờ.")
	}

	if pending.ActionType == model.PendingActionAwaitUtility {
		return s.handleAwaitUtilityCommand(ctx, managerID, chatID, webhookCtx, parsed, pending)
	}

	switch pending.ActionType {
	case model.PendingActionConfirmOverwrite:
		if parsed.Type != CommandConfirm {
			return true, s.sendTextMessage(ctx, managerID, chatID, "Vui lòng nhắn '#ok' để xác nhận hoặc '#huy' để hủy.")
		}
		if err := s.pendingRepo.DeleteByID(ctx, pending.ID); err != nil {
			return true, err
		}
		return true, s.processPendingData(ctx, managerID, webhookCtx, pending.PendingData, stringFromPending(pending.PendingData, "period"), true)
	case model.PendingActionAwaitPeriod:
		if parsed.Type == CommandConfirm {
			if period := onlyPeriodFromPendingOptions(pending.PendingData); period != "" {
				if err := s.pendingRepo.DeleteByID(ctx, pending.ID); err != nil {
					return true, err
				}
				return true, s.processPendingData(ctx, managerID, webhookCtx, pending.PendingData, period, false)
			}
		}

		if parsed.Type != CommandPeriodSelect {
			return true, s.sendTextMessage(ctx, managerID, chatID, "Vui lòng nhắn '#<số tháng>' theo lựa chọn đã gửi, hoặc '#huy' để hủy.")
		}
		period := periodFromPendingOptions(pending.PendingData, parsed.PeriodMonth)
		if period == "" {
			return true, s.sendTextMessage(ctx, managerID, chatID, "Tháng bạn chọn không nằm trong các lựa chọn hiện tại.")
		}
		if err := s.pendingRepo.DeleteByID(ctx, pending.ID); err != nil {
			return true, err
		}
		return true, s.processPendingData(ctx, managerID, webhookCtx, pending.PendingData, period, false)
	default:
		return false, nil
	}
}

// handleAwaitUtilityCommand completes a saved single-room utility reminder with its original period.
func (s *zaloInvoiceCommandServiceImpl) handleAwaitUtilityCommand(ctx context.Context, managerID, chatID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, pending *model.PendingInvoiceUpdate) (bool, error) {
	expectedUtility := stringFromPending(pending.PendingData, "utility_type")
	period := stringFromPending(pending.PendingData, "period")
	if expectedUtility == "" {
		return false, nil
	}

	if parsed.Type == CommandUnknown {
		return false, nil
	}
	if parsed.Type != CommandUtilitySingle || parsed.UtilityType != expectedUtility {
		return true, s.sendTextMessage(ctx, managerID, chatID, awaitUtilityPromptMessage(expectedUtility, period))
	}

	roomID := stringFromPending(pending.PendingData, "room_id")
	room, err := s.roomRepo.GetRoomByIDOnly(ctx, roomID)
	if err != nil {
		return true, err
	}
	return true, s.processAwaitUtilitySingleCommand(ctx, managerID, webhookCtx, parsed, pending.ID, *room, period)
}

// processPendingData resumes a saved single or batch utility command.
func (s *zaloInvoiceCommandServiceImpl) processPendingData(ctx context.Context, managerID string, webhookCtx webhookMessageContext, pendingData map[string]any, forcedPeriod string, allowOverwrite bool) error {
	scope := stringFromPending(pendingData, "command_scope")
	utilityType := stringFromPending(pendingData, "utility_type")
	switch scope {
	case "single":
		roomID := stringFromPending(pendingData, "room_id")
		room, err := s.roomRepo.GetRoomByIDOnly(ctx, roomID)
		if err != nil {
			return err
		}
		parsed := &ParsedCommand{
			Type:        CommandUtilitySingle,
			UtilityType: utilityType,
			Entries: []RoomUtilityEntry{{
				NewIndex:    intFromPending(pendingData, "new_index"),
				HasNewIndex: boolFromPending(pendingData, "has_new_index"),
			}},
		}
		return s.processSingleRoom(ctx, managerID, webhookCtx, parsed, *room, forcedPeriod, allowOverwrite)
	case "batch":
		parsed := &ParsedCommand{
			Type:        CommandUtilityBatch,
			UtilityType: utilityType,
			HouseCode:   stringFromPending(pendingData, "house_code"),
			Entries:     entriesFromPending(pendingData),
		}
		return s.handleBatchCommand(ctx, managerID, webhookCtx, parsed, forcedPeriod, allowOverwrite)
	default:
		return nil
	}
}

// createPeriodPending saves a command until the user chooses the invoice month.
func (s *zaloInvoiceCommandServiceImpl) createPeriodPending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, req utilityUpdateRequest, options map[string]string) error {
	data := map[string]any{
		"command_scope":  "single",
		"utility_type":   req.UtilityType,
		"new_index":      req.NewIndex,
		"has_new_index":  req.HasNewIndex,
		"room_id":        req.Room.ID,
		"period_options": options,
	}
	return s.replacePending(ctx, managerID, webhookCtx, req.Room.ID, model.PendingActionAwaitPeriod, data)
}

// createOverwritePending saves a command until the user confirms overwriting a reading.
func (s *zaloInvoiceCommandServiceImpl) createOverwritePending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, req utilityUpdateRequest, period string, existingInvoice *model.Invoice) error {
	data := map[string]any{
		"command_scope": "single",
		"utility_type":  req.UtilityType,
		"new_index":     req.NewIndex,
		"has_new_index": req.HasNewIndex,
		"period":        period,
		"room_id":       req.Room.ID,
		"old_value":     existingUtilityValue(existingInvoice, req.UtilityType),
	}
	return s.replacePending(ctx, managerID, webhookCtx, req.Room.ID, model.PendingActionConfirmOverwrite, data)
}

// createBatchPeriodPending saves a batch command until the manager chooses a month.
func (s *zaloInvoiceCommandServiceImpl) createBatchPeriodPending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand, options map[string]string) error {
	data := batchPendingData(parsed)
	data["period_options"] = options
	return s.replacePending(ctx, managerID, webhookCtx, "", model.PendingActionAwaitPeriod, data)
}

// createBatchOverwritePending saves a batch command until the manager confirms overwrites.
func (s *zaloInvoiceCommandServiceImpl) createBatchOverwritePending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, parsed *ParsedCommand) error {
	return s.replacePending(ctx, managerID, webhookCtx, "", model.PendingActionConfirmOverwrite, batchPendingData(parsed))
}

// createAwaitUtilityPending saves a non-blocking reminder for a missing utility reading.
func (s *zaloInvoiceCommandServiceImpl) createAwaitUtilityPending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, req utilityUpdateRequest, result utilityUpdateResult) error {
	data := map[string]any{
		"command_scope":     "single",
		"utility_type":      result.MissingUtility,
		"room_id":           req.Room.ID,
		"period":            result.Period,
		"current_utility":   req.UtilityType,
		"current_new_index": result.NewIndex,
	}
	return s.replacePending(ctx, managerID, webhookCtx, req.Room.ID, model.PendingActionAwaitUtility, data)
}

// replacePending replaces the current chat pending state with one new state.
func (s *zaloInvoiceCommandServiceImpl) replacePending(ctx context.Context, managerID string, webhookCtx webhookMessageContext, roomID string, actionType string, data map[string]any) error {
	chatID := commandChatID(webhookCtx)
	if err := s.pendingRepo.DeleteByChatID(ctx, managerID, chatID); err != nil {
		return err
	}
	var roomIDPtr *string
	if roomID != "" {
		roomIDPtr = &roomID
	}
	pending := &model.PendingInvoiceUpdate{
		ManagerID:   managerID,
		ChatID:      chatID,
		IsGroupChat: webhookCtx.isGroupChat,
		RoomID:      roomIDPtr,
		ActionType:  actionType,
		PendingData: data,
	}
	return s.pendingRepo.Create(ctx, pending)
}

// periodFromPendingOptions maps a selected month number to the saved period.
func periodFromPendingOptions(data map[string]any, month int) string {
	optionsRaw, ok := data["period_options"].(map[string]any)
	if !ok {
		if optionsString, ok := data["period_options"].(map[string]string); ok {
			return optionsString[strconv.Itoa(month)]
		}
		return ""
	}
	value, _ := optionsRaw[strconv.Itoa(month)].(string)
	return value
}

// onlyPeriodFromPendingOptions returns the single period if there is exactly one option.
func onlyPeriodFromPendingOptions(data map[string]any) string {
	if optionsRaw, ok := data["period_options"].(map[string]any); ok && len(optionsRaw) == 1 {
		for _, v := range optionsRaw {
			if str, ok := v.(string); ok {
				return str
			}
		}
	}
	if optionsStr, ok := data["period_options"].(map[string]string); ok && len(optionsStr) == 1 {
		for _, v := range optionsStr {
			return v
		}
	}
	return ""
}

// stringFromPending reads a string value from pending JSON data.
func stringFromPending(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

// intFromPending reads an int value from pending JSON data.
func intFromPending(data map[string]any, key string) int {
	switch value := data[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case jsonNumber:
		number, _ := strconv.Atoi(string(value))
		return number
	default:
		return 0
	}
}

type jsonNumber string

// boolFromPending reads a bool value from pending JSON data.
func boolFromPending(data map[string]any, key string) bool {
	value, _ := data[key].(bool)
	return value
}

// entriesFromPending reads batch room entries from pending JSON data.
func entriesFromPending(data map[string]any) []RoomUtilityEntry {
	rawEntries, ok := data["entries"].([]any)
	if !ok {
		return nil
	}

	entries := make([]RoomUtilityEntry, 0, len(rawEntries))
	for _, rawEntry := range rawEntries {
		entryMap, ok := rawEntry.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, RoomUtilityEntry{
			RoomName:    stringFromPending(entryMap, "room_name"),
			NewIndex:    intFromPending(entryMap, "new_index"),
			HasNewIndex: boolFromPending(entryMap, "has_new_index"),
		})
	}
	return entries
}

// batchPendingData serializes a manager batch command into pending JSON data.
func batchPendingData(parsed *ParsedCommand) map[string]any {
	return map[string]any{
		"command_scope": "batch",
		"utility_type":  parsed.UtilityType,
		"house_code":    parsed.HouseCode,
		"entries":       entriesToPendingData(parsed.Entries),
	}
}

// entriesToPendingData serializes room entries for pending JSON storage.
func entriesToPendingData(entries []RoomUtilityEntry) []map[string]any {
	data := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		data = append(data, map[string]any{
			"room_name":     entry.RoomName,
			"new_index":     entry.NewIndex,
			"has_new_index": entry.HasNewIndex,
		})
	}
	return data
}
