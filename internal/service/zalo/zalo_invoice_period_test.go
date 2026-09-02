package zalo

import (
	"context"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// periodOffset returns the yyyy-mm period offset whole months from today.
func periodOffset(offset int) string {
	return shiftMonth(time.Now(), offset).Format(invoicePeriodLayout)
}

// usageHouse is a house billing both utilities by meter reading.
func usageHouse() *model.House {
	return &model.House{ID: "house-1", Name: "679 Quang Trung", HouseCode: "679qt", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}
}

// TestResolvePeriodFollowsRegularHistory verifies a room invoiced every month bills the current
// month without asking the user to pick one.
func TestResolvePeriodFollowsRegularHistory(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", periodOffset(0)).Return(nil, model.ErrInvoiceNotFound)
	for offset := 1; offset <= regularInvoiceHistoryMonths; offset++ {
		period := periodOffset(-offset)
		invoiceRepository.EXPECT().
			GetInvoiceByRoomAndPeriod(ctx, "room-1", period).
			Return(&model.Invoice{ID: "inv-" + period, Period: period, OldElectricityIndex: 100, NewElectricityIndex: 200, OldWaterIndex: 10, NewWaterIndex: 20}, nil)
	}

	service := &zaloInvoiceCommandServiceImpl{invoiceRepo: invoiceRepository}
	resolution, err := service.resolvePeriod(ctx, "room-1", "dien", usageHouse(), "")

	require.NoError(t, err)
	assert.False(t, resolution.NeedsSelection)
	assert.Equal(t, periodOffset(0), resolution.Period)
}

// TestResolvePeriodCompletesOpenInvoice verifies a later reading lands on the invoice that is still
// missing it instead of opening another month.
func TestResolvePeriodCompletesOpenInvoice(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	current := periodOffset(0)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", current).
		Return(&model.Invoice{
			ID:                  "inv-current",
			Period:              current,
			Status:              model.InvoiceStatusUnpaid,
			OldElectricityIndex: 600,
			NewElectricityIndex: 661,
			OldWaterIndex:       100,
			NewWaterIndex:       100,
		}, nil)

	service := &zaloInvoiceCommandServiceImpl{invoiceRepo: invoiceRepository}
	resolution, err := service.resolvePeriod(ctx, "room-1", "nuoc", usageHouse(), "")

	require.NoError(t, err)
	assert.False(t, resolution.NeedsSelection)
	assert.Equal(t, current, resolution.Period)
}

// TestResolvePeriodAsksOnIrregularHistory verifies a gap in the history still lets the user choose.
func TestResolvePeriodAsksOnIrregularHistory(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).
		AnyTimes()

	service := &zaloInvoiceCommandServiceImpl{invoiceRepo: invoiceRepository}
	resolution, err := service.resolvePeriod(ctx, "room-1", "dien", usageHouse(), "")

	require.NoError(t, err)
	assert.True(t, resolution.NeedsSelection)
	assert.Len(t, resolution.Options, 3)
}

// TestResolvePeriodKeepsForcedPeriod verifies a resumed command never re-resolves its month.
func TestResolvePeriodKeepsForcedPeriod(t *testing.T) {
	service := &zaloInvoiceCommandServiceImpl{}
	resolution, err := service.resolvePeriod(context.Background(), "room-1", "dien", usageHouse(), "2026-05")

	require.NoError(t, err)
	assert.Equal(t, "2026-05", resolution.Period)
}

// TestShiftMonthSkipsNoMonthOnLongMonths verifies month arithmetic ignores differing month lengths.
func TestShiftMonthSkipsNoMonthOnLongMonths(t *testing.T) {
	base := time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, "2026-02", shiftMonth(base, -1).Format(invoicePeriodLayout))
	assert.Equal(t, "2026-04", shiftMonth(base, 1).Format(invoicePeriodLayout))
}

// TestInvoiceMissesUtility verifies only usage-billed utilities without a reading count as missing.
func TestInvoiceMissesUtility(t *testing.T) {
	invoice := &model.Invoice{OldElectricityIndex: 100, NewElectricityIndex: 100, OldWaterIndex: 10, NewWaterIndex: 20}
	assert.True(t, invoiceMissesUtility(invoice, "dien", usageHouse()))
	assert.False(t, invoiceMissesUtility(invoice, "nuoc", usageHouse()))
	assert.False(t, invoiceMissesUtility(invoice, "dien", &model.House{ElectricityBillingType: "FIXED", WaterBillingType: "USAGE"}))
}

// TestHandleRoomTargetCommand_ManagerSingleHouse verifies "#dien P201 661" resolves the house when
// the manager owns exactly one, and creates the invoice without any follow-up question.
func TestHandleRoomTargetCommand_ManagerSingleHouse(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().ListHouseByManagerID(ctx, "manager-1", 2, 0, "").Return([]model.House{*usageHouse()}, nil)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(usageHouse(), nil).AnyTimes()
	roomRepository.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", HouseID: "house-1", Name: "P201"}}, nil)

	// Regular monthly history so the current month is billed with no period question.
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", periodOffset(0)).Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	for offset := 1; offset <= regularInvoiceHistoryMonths; offset++ {
		period := periodOffset(-offset)
		invoiceRepository.EXPECT().
			GetInvoiceByRoomAndPeriod(ctx, "room-1", period).
			Return(&model.Invoice{ID: "inv-" + period, Period: period, OldElectricityIndex: 500, NewElectricityIndex: 600, OldWaterIndex: 90, NewWaterIndex: 100}, nil).
			AnyTimes()
	}
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", periodOffset(0)).
		Return(&model.Invoice{NewElectricityIndex: 600, NewWaterIndex: 100}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepository,
		roomRepo:       roomRepository,
		houseRepo:      houseRepository,
		userRepo:       userRepository,
		pendingRepo:    pendingRepository,
		zaloClient:     zaloClient,
		encryptionKey:  encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "sender-1",
		replyChatID: "sender-1",
		senderID:    "sender-1",
		text:        "#dien P201 661",
	})

	require.NoError(t, err)
	assert.Equal(t, periodOffset(0), invoiceService.input.Period)
	assert.Equal(t, 661, invoiceService.input.NewElectricityIndex)
	assert.Equal(t, "room-1", invoiceService.input.RoomID)
	// Water is still missing, so the reminder is the only pending state created.
	require.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitUtility, pendingRepository.created[0].ActionType)
}

// TestHandleRoomTargetCommand_ManagerWithManyHousesNeedsCode verifies the short syntax is refused
// with a hint when the house cannot be inferred.
func TestHandleRoomTargetCommand_ManagerWithManyHousesNeedsCode(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	zaloClient := &recordingZaloClient{}

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().
		ListHouseByManagerID(ctx, "manager-1", 2, 0, "").
		Return([]model.House{{ID: "house-1"}, {ID: "house-2"}}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		houseRepo:     houseRepository,
		userRepo:      userRepository,
		pendingRepo:   &memoryPendingInvoiceUpdateRepository{},
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "sender-1",
		replyChatID: "sender-1",
		senderID:    "sender-1",
		text:        "#dien P201 661",
	})

	require.NoError(t, err)
	require.Len(t, zaloClient.messages, 1)
	assert.Contains(t, zaloClient.messages[0], "mã nhà")
}

// TestHandleRoomTargetCommand_UnknownHouseCodeFallsBackToRoomName verifies an unmatched first token
// is retried as part of a multi-word room name for a single-house manager.
func TestHandleRoomTargetCommand_UnknownHouseCodeFallsBackToRoomName(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().ListHouseByManagerID(ctx, "manager-1", 2, 0, "").Return([]model.House{*usageHouse()}, nil)
	houseRepository.EXPECT().GetHouseByCode(ctx, "manager-1", "phong").Return(nil, model.ErrHouseNotFound)
	roomRepository.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", HouseID: "house-1", Name: "Phòng 201"}}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		houseRepo:     houseRepository,
		roomRepo:      roomRepository,
		userRepo:      userRepository,
		encryptionKey: encryptionKey,
	}

	room, err := service.resolveRoomTargetRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender-1"}, ParseCommand("#dien Phòng 201 661"))

	require.NoError(t, err)
	assert.Equal(t, "room-1", room.ID)
}

// TestResolveRoomTargetRoom_TenantOtherRoomRejected verifies a tenant may only name their own room.
func TestResolveRoomTargetRoom_TenantOtherRoomRejected(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)
	tenantRepository.EXPECT().GetFirstTenantByUserID(ctx, "manager-1", "user-1").Return(&model.FullInfoTenant{RoomID: "room-1"}, nil)
	roomRepository.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", Name: "P101"}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		roomRepo:   roomRepository,
		userRepo:   userRepository,
		tenantRepo: tenantRepository,
	}

	_, err := service.resolveRoomTargetRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender-1"}, ParseCommand("#dien P201 661"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "P101")
}

// TestResolveRoomTargetRoom_TenantOwnRoomAccepted verifies a tenant may confirm their own room name.
func TestResolveRoomTargetRoom_TenantOwnRoomAccepted(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)
	tenantRepository.EXPECT().GetFirstTenantByUserID(ctx, "manager-1", "user-1").Return(&model.FullInfoTenant{RoomID: "room-1"}, nil)
	roomRepository.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", Name: "Phòng 201"}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		roomRepo:   roomRepository,
		userRepo:   userRepository,
		tenantRepo: tenantRepository,
	}

	room, err := service.resolveRoomTargetRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender-1"}, ParseCommand("#dien P201 661"))

	require.NoError(t, err)
	assert.Equal(t, "room-1", room.ID)
}

// TestHandleAwaitUtilityCommand_RoomTargetSupersedesReminder verifies a named room command clears a
// pending reminder and is handled as a fresh command.
func TestHandleAwaitUtilityCommand_RoomTargetSupersedesReminder(t *testing.T) {
	ctx := context.Background()

	pendingRepository := &memoryPendingInvoiceUpdateRepository{
		pending: &model.PendingInvoiceUpdate{
			ID:         "pending-1",
			ManagerID:  "manager-1",
			ChatID:     "chat-1",
			ActionType: model.PendingActionAwaitUtility,
			PendingData: map[string]any{
				"utility_type": "nuoc",
				"room_id":      "room-1",
				"period":       "2026-05",
			},
		},
	}
	service := &zaloInvoiceCommandServiceImpl{pendingRepo: pendingRepository}

	handled, err := service.handleAwaitUtilityCommand(ctx, "manager-1", "chat-1", webhookMessageContext{senderID: "sender-1"}, ParseCommand("#nuoc 679qt P201 123"), pendingRepository.pending)

	require.NoError(t, err)
	assert.False(t, handled)
	assert.Equal(t, "pending-1", pendingRepository.deletedID)
}

// TestHandleInvoiceCommand_SupplementsOpenInvoiceWithoutPending verifies the second utility sent
// days later still lands on the same open invoice even after the reminder state is gone.
func TestHandleInvoiceCommand_SupplementsOpenInvoiceWithoutPending(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	current := periodOffset(0)

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	zaloClient := &recordingZaloClient{}

	roomRepository.EXPECT().GetRoomByGroupChatID(ctx, "chat-group-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"}, nil)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(usageHouse(), nil).AnyTimes()
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

	// Electricity was recorded earlier this month; water is still open on the same invoice.
	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", current).
		Return(&model.Invoice{
			ID:                  "inv-current",
			RoomID:              "room-1",
			Period:              current,
			Status:              model.InvoiceStatusUnpaid,
			OldElectricityIndex: 600,
			NewElectricityIndex: 661,
			OldWaterIndex:       100,
			NewWaterIndex:       100,
		}, nil).
		AnyTimes()
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", current).
		Return(&model.Invoice{NewElectricityIndex: 600, NewWaterIndex: 100}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepository,
		roomRepo:       roomRepository,
		houseRepo:      houseRepository,
		userRepo:       userRepository,
		pendingRepo:    &memoryPendingInvoiceUpdateRepository{},
		zaloClient:     zaloClient,
		encryptionKey:  encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "chat-group-1",
		replyChatID: "chat-group-1",
		senderID:    "sender-1",
		isGroupChat: true,
		text:        "#nuoc 123",
	})

	require.NoError(t, err)
	assert.Equal(t, current, invoiceService.input.Period)
	assert.Equal(t, 123, invoiceService.input.NewWaterIndex)
	assert.Equal(t, 661, invoiceService.input.NewElectricityIndex)
}

// TestFollowUpRoomTarget verifies the follow-up hint repeats the room only when the command named it.
func TestFollowUpRoomTarget(t *testing.T) {
	assert.Equal(t, "679qt p201", followUpRoomTarget(ParseCommand("#dien 679qt P201 661")))
	assert.Equal(t, "p201", followUpRoomTarget(ParseCommand("#dien P201 661")))
	assert.Equal(t, "", followUpRoomTarget(ParseCommand("#dien 661")))
}
