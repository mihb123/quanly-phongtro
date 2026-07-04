package zalo

import (
	"context"
	"testing"

	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type memoryPendingInvoiceUpdateRepository struct {
	pending   *model.PendingInvoiceUpdate
	deletedID string
	created   []*model.PendingInvoiceUpdate
}

// Create records a pending state created by the command service.
func (r *memoryPendingInvoiceUpdateRepository) Create(ctx context.Context, pending *model.PendingInvoiceUpdate) error {
	r.created = append(r.created, pending)
	return nil
}

// GetByChatID returns the in-memory pending state for the test chat.
func (r *memoryPendingInvoiceUpdateRepository) GetByChatID(ctx context.Context, managerID, chatID string) (*model.PendingInvoiceUpdate, error) {
	if r.pending == nil {
		return nil, model.ErrPendingInvoiceUpdateNotFound
	}
	return r.pending, nil
}

// DeleteByChatID clears the in-memory pending state for replacement flows.
func (r *memoryPendingInvoiceUpdateRepository) DeleteByChatID(ctx context.Context, managerID, chatID string) error {
	r.pending = nil
	return nil
}

// DeleteByID records the deleted pending ID.
func (r *memoryPendingInvoiceUpdateRepository) DeleteByID(ctx context.Context, id string) error {
	r.deletedID = id
	return nil
}

type recordingInvoiceService struct {
	invoicesvc.InvoiceService
	input invoicesvc.CreateInvoiceInput
}

// CreateInvoice records the input and returns a completed invoice for command tests.
func (s *recordingInvoiceService) CreateInvoice(ctx context.Context, managerID string, input invoicesvc.CreateInvoiceInput) (*model.InvoiceWithRoom, error) {
	s.input = input
	oldWaterIndex := 0
	if input.OldWaterIndex != nil {
		oldWaterIndex = *input.OldWaterIndex
	}
	oldElectricityIndex := 0
	if input.OldElectricityIndex != nil {
		oldElectricityIndex = *input.OldElectricityIndex
	}
	return &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:                  "invoice-1",
			RoomID:              input.RoomID,
			Period:              input.Period,
			OldElectricityIndex: oldElectricityIndex,
			NewElectricityIndex: input.NewElectricityIndex,
			OldWaterIndex:       oldWaterIndex,
			NewWaterIndex:       input.NewWaterIndex,
			TotalAmount:         1500000,
		},
		RoomName: "P101",
		HouseID:  "house-1",
	}, nil
}

type recordingZaloClient struct {
	ZaloClient
	messages []string
}

// SendMessage records outgoing command responses.
func (c *recordingZaloClient) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	c.messages = append(c.messages, text)
	return nil
}

// encryptedToken returns a bot token encrypted with the test key.
func encryptedToken(t *testing.T, encryptionKey []byte) string {
	t.Helper()
	token, err := security.Encrypt("bot-token", encryptionKey)
	require.NoError(t, err)
	return token
}

// TestHandleInvoiceCommand_AwaitUtilityUsesSavedPeriod verifies follow-up readings do not advance to a new invoice period.
func TestHandleInvoiceCommand_AwaitUtilityUsesSavedPeriod(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
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
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	roomRepository.EXPECT().
		GetRoomByIDOnly(ctx, "room-1").
		Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P101"}, nil)
	houseRepository.EXPECT().
		GetByID(ctx, "house-1", "manager-1").
		Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").
		Return(&model.Invoice{
			ID:                  "existing-invoice",
			RoomID:              "room-1",
			Period:              "2026-05",
			OldElectricityIndex: 100,
			NewElectricityIndex: 150,
			OldWaterIndex:       110,
			NewWaterIndex:       110,
		}, nil)
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", "2026-05").
		Return(nil, model.ErrInvoiceNotFound)
	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil)

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
		chatID:   "chat-1",
		senderID: "sender-1",
		text:     "#nuoc 120",
	})

	require.NoError(t, err)
	require.NotNil(t, invoiceService.input.OldWaterIndex)
	assert.Equal(t, "2026-05", invoiceService.input.Period)
	assert.Equal(t, 110, *invoiceService.input.OldWaterIndex)
	assert.Equal(t, 120, invoiceService.input.NewWaterIndex)
	assert.Equal(t, "pending-1", pendingRepository.deletedID)
}

// TestHandleInvoiceCommand_AwaitUtilityRejectsWrongUtility verifies wrong follow-up commands keep the saved pending state.
func TestHandleInvoiceCommand_AwaitUtilityRejectsWrongUtility(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepository := mock_model.NewMockUserRepository(ctrl)
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
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		userRepo:      userRepository,
		pendingRepo:   pendingRepository,
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:   "chat-1",
		senderID: "sender-1",
		text:     "#dien 180",
	})

	require.NoError(t, err)
	assert.Empty(t, pendingRepository.deletedID)
	require.Len(t, zaloClient.messages, 1)
	assert.Contains(t, zaloClient.messages[0], "#nuoc <số mới>")
	assert.Contains(t, zaloClient.messages[0], "05/2026")
}

// TestHandleInvoiceCommand_AwaitUtilityKeepsPendingOnRetryableError verifies missing readings do not clear period context.
func TestHandleInvoiceCommand_AwaitUtilityKeepsPendingOnRetryableError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
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
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	roomRepository.EXPECT().
		GetRoomByIDOnly(ctx, "room-1").
		Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P101"}, nil)
	houseRepository.EXPECT().
		GetByID(ctx, "house-1", "manager-1").
		Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		roomRepo:      roomRepository,
		houseRepo:     houseRepository,
		userRepo:      userRepository,
		pendingRepo:   pendingRepository,
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:   "chat-1",
		senderID: "sender-1",
		text:     "#nuoc",
	})

	require.NoError(t, err)
	assert.Empty(t, pendingRepository.deletedID)
	require.Len(t, zaloClient.messages, 1)
	assert.Contains(t, zaloClient.messages[0], "vui lòng nhập chỉ số nước mới")
}
