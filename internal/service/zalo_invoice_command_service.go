package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type ZaloInvoiceCommandService interface {
	HandleInvoiceCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext) error
	HasPendingState(ctx context.Context, managerID, chatID string) bool
}

type zaloInvoiceCommandServiceImpl struct {
	invoiceService InvoiceService
	invoiceRepo    model.InvoiceRepository
	roomRepo       model.RoomRepository
	houseRepo      model.HouseRepository
	tenantRepo     model.TenantRepository
	userRepo       model.UserRepository
	pendingRepo    model.PendingInvoiceUpdateRepository
	zaloClient     ZaloClient
	imageService   ImageService
	encryptionKey  []byte
	publicBaseURL  string
}

type utilityUpdateRequest struct {
	UtilityType    string
	Room           model.Room
	NewIndex       int
	HasNewIndex    bool
	ForcedPeriod   string
	AllowOverwrite bool
}

type utilityUpdateResult struct {
	Invoice        *model.InvoiceWithRoom
	RoomName       string
	Period         string
	OldIndex       int
	NewIndex       int
	MissingUtility string
	Complete       bool
}

type periodResolution struct {
	Period         string
	NeedsSelection bool
	Options        map[string]string
}

type matchedBatchEntry struct {
	Entry RoomUtilityEntry
	Room  model.Room
}

// NewZaloInvoiceCommandService creates the business handler for invoice chat commands.
func NewZaloInvoiceCommandService(invoiceService InvoiceService, invoiceRepo model.InvoiceRepository, roomRepo model.RoomRepository, houseRepo model.HouseRepository, tenantRepo model.TenantRepository, userRepo model.UserRepository, pendingRepo model.PendingInvoiceUpdateRepository, zaloClient ZaloClient, imageService ImageService, encryptionKey []byte, publicBaseURL string) ZaloInvoiceCommandService {
	return &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepo,
		roomRepo:       roomRepo,
		houseRepo:      houseRepo,
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		pendingRepo:    pendingRepo,
		zaloClient:     zaloClient,
		imageService:   imageService,
		encryptionKey:  encryptionKey,
		publicBaseURL:  strings.TrimRight(publicBaseURL, "/"),
	}
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

// HandleInvoiceCommand processes a parsed invoice command from a Zalo webhook message.
func (s *zaloInvoiceCommandServiceImpl) HandleInvoiceCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext) error {
	parsed := ParseCommand(webhookCtx.text)
	chatID := commandChatID(webhookCtx)
	if chatID == "" {
		return nil
	}

	pending, err := s.pendingRepo.GetByChatID(ctx, managerID, chatID)
	if err != nil && !errors.Is(err, model.ErrPendingInvoiceUpdateNotFound) {
		return err
	}
	if pending != nil {
		handled, err := s.handlePendingCommand(ctx, managerID, chatID, webhookCtx, parsed, pending)
		if handled || err != nil {
			return err
		}
	}

	switch parsed.Type {
	case CommandUtilitySingle:
		return s.handleSingleCommand(ctx, managerID, webhookCtx, parsed, "", false)
	case CommandUtilityBatch:
		return s.handleBatchCommand(ctx, managerID, webhookCtx, parsed, "", false)
	case CommandConfirm, CommandCancel, CommandPeriodSelect:
		return s.sendTextMessage(ctx, managerID, chatID, "Không có thao tác hóa đơn nào đang chờ xử lý.")
	default:
		return nil
	}
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
	return s.respondAfterSingleUpdate(ctx, managerID, commandChatID(webhookCtx), result)
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
	return s.respondAfterSingleUpdate(ctx, managerID, commandChatID(webhookCtx), result)
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
		resolution, err := s.resolvePeriod(ctx, matchedEntry.Room.ID, forcedPeriod)
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

// applyUtilityUpdate creates or updates one invoice reading and records pending state when needed.
func (s *zaloInvoiceCommandServiceImpl) applyUtilityUpdate(ctx context.Context, managerID string, webhookCtx webhookMessageContext, req utilityUpdateRequest) (utilityUpdateResult, bool, error) {
	chatID := commandChatID(webhookCtx)
	house, err := s.houseRepo.GetByID(ctx, req.Room.HouseID, managerID)
	if err != nil {
		return utilityUpdateResult{}, false, err
	}

	if utilityNeedsReading(req.UtilityType, house) && !req.HasNewIndex {
		return utilityUpdateResult{}, false, fmt.Errorf("vui lòng nhập chỉ số %s mới", utilityLabel(req.UtilityType))
	}

	resolution, err := s.resolvePeriod(ctx, req.Room.ID, req.ForcedPeriod)
	if err != nil {
		return utilityUpdateResult{}, false, err
	}
	if resolution.NeedsSelection {
		if err := s.createPeriodPending(ctx, managerID, webhookCtx, req, resolution.Options); err != nil {
			return utilityUpdateResult{}, false, err
		}
		return utilityUpdateResult{}, true, s.sendTextMessage(ctx, managerID, chatID, periodSelectionMessage(req.Room.Name, resolution.Options))
	}

	existingInvoice, err := s.invoiceRepo.GetInvoiceByRoomAndPeriod(ctx, req.Room.ID, resolution.Period)
	if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
		return utilityUpdateResult{}, false, err
	}
	if existingInvoice != nil && shouldConfirmOverwrite(existingInvoice, house, req) {
		if !req.AllowOverwrite {
			if err := s.createOverwritePending(ctx, managerID, webhookCtx, req, resolution.Period, existingInvoice); err != nil {
				return utilityUpdateResult{}, false, err
			}
			message := overwriteConfirmMessage(resolution.Period, req.UtilityType, existingUtilityValue(existingInvoice, req.UtilityType), req.NewIndex)
			return utilityUpdateResult{}, true, s.sendTextMessage(ctx, managerID, chatID, message)
		}
	}

	input, oldIndex, err := s.buildInvoiceInput(ctx, req, resolution.Period, existingInvoice)
	if err != nil {
		return utilityUpdateResult{}, false, err
	}
	invoice, err := s.invoiceService.CreateInvoice(ctx, managerID, input)
	if err != nil {
		return utilityUpdateResult{}, false, err
	}

	result := utilityUpdateResult{
		Invoice:        invoice,
		RoomName:       invoice.RoomName,
		Period:         invoice.Period,
		OldIndex:       oldIndex,
		NewIndex:       newUtilityValue(invoice, req.UtilityType),
		MissingUtility: missingUtilityType(invoice, house),
	}
	result.Complete = result.MissingUtility == ""

	if !result.Complete {
		if err := s.createAwaitUtilityPending(ctx, managerID, webhookCtx, req, result); err != nil {
			return utilityUpdateResult{}, false, err
		}
	}

	return result, false, nil
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
		return nil, errors.New("Quản lý vui lòng dùng cú pháp nhiều phòng: #dien <mã nhà>")
	}

	tenant, err := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, linkedUser.ID)
	if err != nil {
		return nil, errors.New("Không tìm thấy phòng đang thuê cho tài khoản này.")
	}
	return s.roomRepo.GetRoomByIDOnly(ctx, tenant.RoomID)
}

// ensureManagerSender verifies that a private batch command came from the linked manager.
func (s *zaloInvoiceCommandServiceImpl) ensureManagerSender(ctx context.Context, managerID, senderID string) error {
	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, senderID)
	if err != nil {
		return errors.New("Tài khoản Zalo này chưa liên kết với hệ thống.")
	}
	if linkedUser.ID != managerID || linkedUser.Role != model.RoleManager {
		return errors.New("Chỉ quản lý mới được cập nhật nhiều phòng qua chat riêng.")
	}
	return nil
}

// resolvePeriod returns the forced, current active, next, or user-selected invoice period.
func (s *zaloInvoiceCommandServiceImpl) resolvePeriod(ctx context.Context, roomID, forcedPeriod string) (periodResolution, error) {
	if forcedPeriod != "" {
		return periodResolution{Period: forcedPeriod}, nil
	}

	latestInvoice, err := s.invoiceRepo.GetLatestInvoiceByRoomID(ctx, roomID)
	if err != nil {
		if errors.Is(err, model.ErrInvoiceNotFound) {
			return periodResolution{NeedsSelection: true, Options: currentPeriodOptions(time.Now())}, nil
		}
		return periodResolution{}, err
	}

	currentPeriod := time.Now().Format("2006-01")
	if latestInvoice.Period >= currentPeriod {
		return periodResolution{Period: latestInvoice.Period}, nil
	}

	nextPeriod, err := addMonthToPeriod(latestInvoice.Period)
	if err != nil {
		return periodResolution{}, err
	}
	return periodResolution{Period: nextPeriod}, nil
}

// buildInvoiceInput preserves existing or previous invoice values around one changed utility.
func (s *zaloInvoiceCommandServiceImpl) buildInvoiceInput(ctx context.Context, req utilityUpdateRequest, period string, existingInvoice *model.Invoice) (CreateInvoiceInput, int, error) {
	previousInvoice, err := s.invoiceRepo.GetPreviousInvoice(ctx, req.Room.ID, period)
	if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
		return CreateInvoiceInput{}, 0, err
	}

	oldElectricityIndex := 0
	oldWaterIndex := 0
	newElectricityIndex := 0
	newWaterIndex := 0
	otherFee := 0.0
	discount := 0.0
	vehicleCount := 0
	var tenantCount *int

	if previousInvoice != nil {
		oldElectricityIndex = previousInvoice.NewElectricityIndex
		oldWaterIndex = previousInvoice.NewWaterIndex
		newElectricityIndex = previousInvoice.NewElectricityIndex
		newWaterIndex = previousInvoice.NewWaterIndex
		otherFee = previousInvoice.OtherFee
		discount = previousInvoice.Discount
		vehicleCount = previousInvoice.VehicleCount
	}

	if existingInvoice != nil {
		oldElectricityIndex = existingInvoice.OldElectricityIndex
		oldWaterIndex = existingInvoice.OldWaterIndex
		newElectricityIndex = existingInvoice.NewElectricityIndex
		newWaterIndex = existingInvoice.NewWaterIndex
		otherFee = existingInvoice.OtherFee
		discount = existingInvoice.Discount
		vehicleCount = existingInvoice.VehicleCount
		count := existingInvoice.TenantCount
		tenantCount = &count
	}

	if req.UtilityType == "dien" && req.HasNewIndex {
		newElectricityIndex = req.NewIndex
	}
	if req.UtilityType == "nuoc" && req.HasNewIndex {
		newWaterIndex = req.NewIndex
	}

	input := CreateInvoiceInput{
		RoomID:              req.Room.ID,
		Period:              period,
		OldElectricityIndex: &oldElectricityIndex,
		NewElectricityIndex: newElectricityIndex,
		OldWaterIndex:       &oldWaterIndex,
		NewWaterIndex:       newWaterIndex,
		OtherFee:            otherFee,
		Discount:            discount,
		VehicleCount:        vehicleCount,
		TenantCount:         tenantCount,
	}

	if req.UtilityType == "dien" {
		return input, oldElectricityIndex, nil
	}
	return input, oldWaterIndex, nil
}

// respondAfterSingleUpdate sends the user-facing result for one room command.
func (s *zaloInvoiceCommandServiceImpl) respondAfterSingleUpdate(ctx context.Context, managerID, chatID string, result utilityUpdateResult) error {
	if !result.Complete {
		period := displayPeriod(result.Period)
		message := fmt.Sprintf("Đã ghi nhận số %s mới: %d cho %s (tháng %s).\nVui lòng bổ sung số %s tháng %s bằng cú pháp: #%s <số mới> hoặc #huy để hủy.", utilityLabelFromMissing(result), result.NewIndex, result.RoomName, period, utilityLabel(result.MissingUtility), period, result.MissingUtility)
		return s.sendTextMessage(ctx, managerID, chatID, message)
	}

	if err := s.deliverInvoice(ctx, managerID, result.Invoice.ID); err != nil {
		_ = s.sendTextMessage(ctx, managerID, chatID, "Đã tạo hóa đơn nhưng chưa gửi được ảnh: "+err.Error())
		return nil
	}
	message := fmt.Sprintf("Hóa đơn tháng %s cho %s. Tổng tiền: %s", result.Period, result.RoomName, formatCurrencyToVND(result.Invoice.TotalAmount))
	return s.sendTextMessage(ctx, managerID, chatID, message)
}

// respondAfterBatchUpdate sends a compact manager summary and photos for completed invoices.
func (s *zaloInvoiceCommandServiceImpl) respondAfterBatchUpdate(ctx context.Context, managerID, chatID string, house model.House, utilityType string, results []utilityUpdateResult, failed []string) error {
	lines := []string{fmt.Sprintf("Đã cập nhật số %s cho nhà %s:", utilityLabel(utilityType), house.Name)}
	needsMissingUtility := false
	for _, result := range results {
		usage := result.NewIndex - result.OldIndex
		lines = append(lines, fmt.Sprintf("- %s: %d (cũ: %d) -> %d số", result.RoomName, result.NewIndex, result.OldIndex, usage))
		if result.Complete {
			if err := s.deliverInvoice(ctx, managerID, result.Invoice.ID); err != nil {
				lines = append(lines, fmt.Sprintf("  Chưa gửi được ảnh: %v", err))
			}
		} else {
			needsMissingUtility = true
		}
	}
	if len(results) > 0 && needsMissingUtility {
		missingUtility := oppositeUtility(utilityType)
		lines = append(lines, fmt.Sprintf("Bổ sung số %s bằng: #%s %s", utilityLabel(missingUtility), missingUtility, house.HouseCode))
	}
	lines = append(lines, failed...)
	return s.sendTextMessage(ctx, managerID, chatID, strings.Join(lines, "\n"))
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

// sendTextMessage sends a plain Zalo message through the manager's bot.
func (s *zaloInvoiceCommandServiceImpl) sendTextMessage(ctx context.Context, managerID, chatID, text string) error {
	_, botToken, err := getDecryptedZaloToken(ctx, s.userRepo, managerID, s.encryptionKey)
	if err != nil {
		return err
	}
	if err := s.zaloClient.SendMessage(ctx, botToken, chatID, text); err != nil {
		if isZaloAuthError(err) {
			inactive := false
			if _, updateErr := s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{IsZaloBotActive: &inactive}); updateErr != nil {
				return fmt.Errorf("mark zalo bot token inactive: %w", updateErr)
			}
			return fmt.Errorf("zalo bot token is invalid or expired")
		}
		return err
	}
	return nil
}

// deliverInvoice sends one completed invoice image through the shared Zalo delivery helper.
func (s *zaloInvoiceCommandServiceImpl) deliverInvoice(ctx context.Context, managerID, invoiceID string) error {
	return deliverInvoiceToZalo(ctx, zaloInvoiceDeliveryDeps{
		client:        s.zaloClient,
		userRepo:      s.userRepo,
		roomRepo:      s.roomRepo,
		tenantRepo:    s.tenantRepo,
		invoiceRepo:   s.invoiceRepo,
		imageService:  s.imageService,
		encryptionKey: s.encryptionKey,
		publicBaseURL: s.publicBaseURL,
		markTokenInactive: func(ctx context.Context, managerID string) {
			inactive := false
			if _, err := s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{IsZaloBotActive: &inactive}); err != nil {
				fmt.Printf("mark zalo bot token inactive: %v\n", err)
			}
		},
	}, managerID, invoiceID)
}

// commandChatID returns the chat ID where command replies should be stored and sent.
func commandChatID(webhookCtx webhookMessageContext) string {
	if webhookCtx.replyChatID != "" {
		return webhookCtx.replyChatID
	}
	if webhookCtx.chatID != "" {
		return webhookCtx.chatID
	}
	return webhookCtx.senderID
}

// isInvoiceCommandText reports whether a message starts an invoice command flow.
func isInvoiceCommandText(text string) bool {
	command := ParseCommand(text)
	return command.Type != CommandUnknown
}

// utilityNeedsReading reports whether a utility command must include a meter index.
func utilityNeedsReading(utilityType string, house *model.House) bool {
	if utilityType == "dien" {
		return house.ElectricityBillingType != "FIXED"
	}
	return house.WaterBillingType != "FIXED"
}

// shouldConfirmOverwrite reports whether a new reading would replace an entered value.
func shouldConfirmOverwrite(invoice *model.Invoice, house *model.House, req utilityUpdateRequest) bool {
	if !utilityNeedsReading(req.UtilityType, house) || !req.HasNewIndex {
		return false
	}
	currentValue := existingUtilityValue(invoice, req.UtilityType)
	oldValue := existingOldUtilityValue(invoice, req.UtilityType)
	return currentValue != oldValue && currentValue != req.NewIndex
}

// missingUtilityType returns the first usage-based utility that still has no entered reading.
func missingUtilityType(invoice *model.InvoiceWithRoom, house *model.House) string {
	if house.ElectricityBillingType != "FIXED" && invoice.NewElectricityIndex == invoice.OldElectricityIndex {
		return "dien"
	}
	if house.WaterBillingType != "FIXED" && invoice.NewWaterIndex == invoice.OldWaterIndex {
		return "nuoc"
	}
	return ""
}

// existingUtilityValue returns the existing new index for a utility.
func existingUtilityValue(invoice *model.Invoice, utilityType string) int {
	if utilityType == "dien" {
		return invoice.NewElectricityIndex
	}
	return invoice.NewWaterIndex
}

// existingOldUtilityValue returns the existing old index for a utility.
func existingOldUtilityValue(invoice *model.Invoice, utilityType string) int {
	if utilityType == "dien" {
		return invoice.OldElectricityIndex
	}
	return invoice.OldWaterIndex
}

// newUtilityValue returns the new index from a completed invoice response.
func newUtilityValue(invoice *model.InvoiceWithRoom, utilityType string) int {
	if utilityType == "dien" {
		return invoice.NewElectricityIndex
	}
	return invoice.NewWaterIndex
}

// utilityLabel returns the Vietnamese display name for a utility type.
func utilityLabel(utilityType string) string {
	if utilityType == "dien" {
		return "điện"
	}
	return "nước"
}

// awaitUtilityPromptMessage tells the user which saved utility command is still expected.
func awaitUtilityPromptMessage(utilityType, period string) string {
	if period == "" {
		return fmt.Sprintf("Đang chờ số %s. Vui lòng nhập #%s <số mới> hoặc #huy để hủy.", utilityLabel(utilityType), utilityType)
	}
	return fmt.Sprintf("Đang chờ số %s tháng %s. Vui lòng nhập #%s <số mới> hoặc #huy để hủy.", utilityLabel(utilityType), displayPeriod(period), utilityType)
}

// utilityLabelFromMissing returns the entered utility label from a single update result.
func utilityLabelFromMissing(result utilityUpdateResult) string {
	if result.MissingUtility == "dien" {
		return "nước"
	}
	return "điện"
}

// oppositeUtility returns the other supported utility type.
func oppositeUtility(utilityType string) string {
	if utilityType == "dien" {
		return "nuoc"
	}
	return "dien"
}

// addMonthToPeriod returns the next yyyy-mm period.
func addMonthToPeriod(period string) (string, error) {
	parsed, err := time.Parse("2006-01", period)
	if err != nil {
		return "", fmt.Errorf("invalid invoice period %q: %w", period, err)
	}
	return parsed.AddDate(0, 1, 0).Format("2006-01"), nil
}

// currentPeriodOptions returns previous and current month choices for rooms without invoices.
func currentPeriodOptions(now time.Time) map[string]string {
	previous := now.AddDate(0, -1, 0)
	return map[string]string{
		strconv.Itoa(int(previous.Month())): previous.Format("2006-01"),
		strconv.Itoa(int(now.Month())):      now.Format("2006-01"),
	}
}

// periodSelectionMessage builds the prompt for rooms without an invoice history.
func periodSelectionMessage(roomName string, options map[string]string) string {
	lines := []string{
		fmt.Sprintf("%s chưa có hóa đơn trước đó.", roomName),
		"Bạn muốn tạo hóa đơn cho tháng nào?",
	}
	for month, period := range options {
		lines = append(lines, fmt.Sprintf("Nhắn '#%s' cho tháng %s", month, displayPeriod(period)))
	}
	return strings.Join(lines, "\n")
}

// overwriteConfirmMessage builds the prompt for replacing an existing utility reading.
func overwriteConfirmMessage(period, utilityType string, oldValue, newValue int) string {
	return fmt.Sprintf("Số %s tháng %s đã được ghi là %d.\nBạn muốn cập nhật thành %d?\nNhắn '#ok' để xác nhận, '#huy' để hủy.", utilityLabel(utilityType), displayPeriod(period), oldValue, newValue)
}

// displayPeriod converts yyyy-mm into mm/yyyy for chat messages.
func displayPeriod(period string) string {
	parsed, err := time.Parse("2006-01", period)
	if err != nil {
		return period
	}
	return parsed.Format("01/2006")
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
