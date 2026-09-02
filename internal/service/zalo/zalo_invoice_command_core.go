package zalo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

const (
	invoicePeriodLayout = "2006-01"
	// regularInvoiceHistoryMonths is how many consecutive prior months of invoices prove that a room
	// is billed every month, so a new reading belongs to the current month with no question asked.
	regularInvoiceHistoryMonths = 3
)

type utilityUpdateRequest struct {
	UtilityType    string
	Room           model.Room
	NewIndex       int
	HasNewIndex    bool
	ForcedPeriod   string
	AllowOverwrite bool
	IsBatch        bool
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

	resolution, err := s.resolvePeriod(ctx, req.Room.ID, req.UtilityType, house, req.ForcedPeriod)
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
		if !req.IsBatch && (errors.Is(err, model.ErrInvalidElectricityIndex) || errors.Is(err, model.ErrInvalidWaterIndex)) {
			data := map[string]any{
				"command_scope": "single",
				"utility_type":  req.UtilityType,
				"room_id":       req.Room.ID,
				"period":        resolution.Period,
			}
			if pErr := s.replacePending(ctx, managerID, webhookCtx, req.Room.ID, model.PendingActionAwaitUtility, data); pErr != nil {
				return utilityUpdateResult{}, false, pErr
			}
			msg := fmt.Sprintf("%s\nVui lòng nhập lại bằng cú pháp: #%s <số đúng> hoặc #huy để hủy.", err.Error(), req.UtilityType)
			return utilityUpdateResult{}, true, s.sendTextMessage(ctx, managerID, chatID, msg)
		}
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

// resolvePeriod decides which month a reading belongs to without asking the user whenever the
// answer is unambiguous: an invoice that is still waiting for this very reading, or the current
// month when the room has been invoiced every month so far. Only a room with an irregular history
// falls back to letting the user pick a month.
func (s *zaloInvoiceCommandServiceImpl) resolvePeriod(ctx context.Context, roomID, utilityType string, house *model.House, forcedPeriod string) (periodResolution, error) {
	if forcedPeriod != "" {
		return periodResolution{Period: forcedPeriod}, nil
	}

	now := time.Now()
	lookup := s.invoiceLookup(ctx, roomID)

	// --- 1. COMPLETE AN OPEN INVOICE ---
	// A reading sent days after its counterpart must land on the same invoice, not open a new month.
	for _, offset := range []int{0, -1} {
		month := shiftMonth(now, offset)
		invoice, err := lookup(month.Format(invoicePeriodLayout))
		if err != nil {
			return periodResolution{}, err
		}
		if invoice != nil && invoice.Status != model.InvoiceStatusPaid && invoiceMissesUtility(invoice, utilityType, house) {
			return periodResolution{Period: invoice.Period}, nil
		}
	}

	// --- 2. FOLLOW A KNOWN MONTHLY CADENCE ---
	regular, err := hasRegularInvoiceHistory(lookup, now)
	if err != nil {
		return periodResolution{}, err
	}
	if regular {
		return periodResolution{Period: shiftMonth(now, 0).Format(invoicePeriodLayout)}, nil
	}

	// --- 3. ASK WHICH MONTH TO BILL ---
	options := make(map[string]string)
	for _, offset := range []int{-1, 0, 1} {
		month := shiftMonth(now, offset)
		invoice, err := lookup(month.Format(invoicePeriodLayout))
		if err != nil {
			return periodResolution{}, err
		}
		if invoice == nil {
			options[strconv.Itoa(int(month.Month()))] = month.Format(invoicePeriodLayout)
		}
	}

	if len(options) == 0 {
		return periodResolution{Period: shiftMonth(now, 0).Format(invoicePeriodLayout)}, nil
	}

	return periodResolution{NeedsSelection: true, Options: options}, nil
}

// invoiceLookup returns a per-command memoized invoice reader so resolving one period never
// queries the same month twice, and reports a missing invoice as a nil invoice instead of an error.
func (s *zaloInvoiceCommandServiceImpl) invoiceLookup(ctx context.Context, roomID string) func(period string) (*model.Invoice, error) {
	cache := make(map[string]*model.Invoice)
	return func(period string) (*model.Invoice, error) {
		if invoice, ok := cache[period]; ok {
			return invoice, nil
		}
		invoice, err := s.invoiceRepo.GetInvoiceByRoomAndPeriod(ctx, roomID, period)
		if err != nil {
			if !errors.Is(err, model.ErrInvoiceNotFound) {
				return nil, err
			}
			invoice = nil
		}
		cache[period] = invoice
		return invoice, nil
	}
}

// hasRegularInvoiceHistory reports whether every month right before now already has an invoice,
// which makes the current month the only sensible target for a new reading.
func hasRegularInvoiceHistory(lookup func(string) (*model.Invoice, error), now time.Time) (bool, error) {
	for offset := 1; offset <= regularInvoiceHistoryMonths; offset++ {
		invoice, err := lookup(shiftMonth(now, -offset).Format(invoicePeriodLayout))
		if err != nil {
			return false, err
		}
		if invoice == nil {
			return false, nil
		}
	}
	return true, nil
}

// shiftMonth moves t by whole months from the first day of its month, so month lengths never make
// the result skip a month (time.AddDate on the 31st would turn "one month back" into the same month).
func shiftMonth(t time.Time, offset int) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).AddDate(0, offset, 0)
}

// invoiceMissesUtility reports whether a usage-billed utility still has no reading on an invoice.
func invoiceMissesUtility(invoice *model.Invoice, utilityType string, house *model.House) bool {
	if !utilityNeedsReading(utilityType, house) {
		return false
	}
	return existingUtilityValue(invoice, utilityType) == existingOldUtilityValue(invoice, utilityType)
}

// buildInvoiceInput preserves existing or previous invoice values around one changed utility.
func (s *zaloInvoiceCommandServiceImpl) buildInvoiceInput(ctx context.Context, req utilityUpdateRequest, period string, existingInvoice *model.Invoice) (invoicesvc.CreateInvoiceInput, int, error) {
	previousInvoice, err := s.invoiceRepo.GetPreviousInvoice(ctx, req.Room.ID, period)
	if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
		return invoicesvc.CreateInvoiceInput{}, 0, err
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

	input := invoicesvc.CreateInvoiceInput{
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

// respondAfterSingleUpdate sends the user-facing result for one room command. followUpTarget is the
// room part the user must repeat to complete the invoice, empty when the chat already implies it.
func (s *zaloInvoiceCommandServiceImpl) respondAfterSingleUpdate(ctx context.Context, managerID, chatID string, result utilityUpdateResult, followUpTarget string) error {
	if !result.Complete {
		period := displayPeriod(result.Period)
		followUpCommand := "#" + result.MissingUtility
		if followUpTarget != "" {
			followUpCommand += " " + followUpTarget
		}
		message := fmt.Sprintf("Đã ghi nhận số %s mới: %d cho %s (tháng %s).\nVui lòng bổ sung số %s tháng %s bằng cú pháp: %s <số mới> hoặc #huy để hủy.", utilityLabelFromMissing(result), result.NewIndex, result.RoomName, period, utilityLabel(result.MissingUtility), period, followUpCommand)
		return s.sendTextMessage(ctx, managerID, chatID, message)
	}

	if err := s.deliverInvoice(ctx, managerID, result.Invoice.ID); err != nil {
		_ = s.sendTextMessage(ctx, managerID, chatID, "Đã tạo hóa đơn nhưng chưa gửi được ảnh: "+err.Error())
		return nil
	}
	return nil
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
		client:         s.zaloClient,
		userRepo:       s.userRepo,
		roomRepo:       s.roomRepo,
		tenantRepo:     s.tenantRepo,
		invoiceRepo:    s.invoiceRepo,
		imageService:   s.imageService,
		paymentService: s.paymentService,
		encryptionKey:  s.encryptionKey,
		publicBaseURL:  s.publicBaseURL,
		markTokenInactive: func(ctx context.Context, managerID string) {
			inactive := false
			if _, err := s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{IsZaloBotActive: &inactive}); err != nil {
				logger.Error(nil, 0, "mark zalo bot token inactive failed", err)
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

// isInvoiceCommandText reports whether a message starts an invoice command flow. Help is excluded
// because the webhook answers it before any invoice state is touched.
func isInvoiceCommandText(text string) bool {
	command := ParseCommand(text)
	return command.Type != CommandUnknown && command.Type != CommandHelp
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

// periodSelectionMessage builds the prompt for period selection.
func periodSelectionMessage(targetName string, options map[string]string) string {
	if len(options) == 1 {
		for _, period := range options {
			return fmt.Sprintf("Bạn muốn tạo hóa đơn tháng %s cho %s đúng không?\nNhắn '#ok' để tiếp tục, hoặc '#huy' để hủy.", displayPeriod(period), targetName)
		}
	}

	lines := []string{
		fmt.Sprintf("Bạn muốn tạo hóa đơn cho tháng nào cho %s?", targetName),
	}
	var months []int
	for mStr := range options {
		if m, err := strconv.Atoi(mStr); err == nil {
			months = append(months, m)
		}
	}
	sort.Ints(months)
	for _, m := range months {
		period := options[strconv.Itoa(m)]
		lines = append(lines, fmt.Sprintf("Nhắn '#%d' cho tháng %s", m, displayPeriod(period)))
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
