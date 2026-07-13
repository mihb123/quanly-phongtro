package zalo

import (
	"context"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleInvoiceCommand_SingleCommandGroupChat(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	// group chat resolution
	roomRepository.EXPECT().
		GetRoomByGroupChatID(ctx, "chat-group-1").
		Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P101"}, nil)
	houseRepository.EXPECT().
		GetByID(ctx, "house-1", "manager-1").
		Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil).AnyTimes()

	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

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
		chatID:      "chat-group-1",
		senderID:    "sender-1",
		isGroupChat: true,
		text:        "#dien 200",
	})

	require.NoError(t, err)
	assert.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitPeriod, pendingRepository.created[0].ActionType)
}

func TestHandleInvoiceCommand_SingleCommandPrivateChat(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	// private chat resolution
	userRepository.EXPECT().
		GetByZaloUserID(ctx, "sender-1").
		Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)

	tenantRepository.EXPECT().
		GetFirstTenantByUserID(ctx, "manager-1", "user-1").
		Return(&model.FullInfoTenant{RoomID: "room-1"}, nil)

	roomRepository.EXPECT().
		GetRoomByIDOnly(ctx, "room-1").
		Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P101"}, nil)
	houseRepository.EXPECT().
		GetByID(ctx, "house-1", "manager-1").
		Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, "room-1", gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepository,
		roomRepo:       roomRepository,
		houseRepo:      houseRepository,
		userRepo:       userRepository,
		tenantRepo:     tenantRepository,
		pendingRepo:    pendingRepository,
		zaloClient:     zaloClient,
		encryptionKey:  encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "sender-1",
		senderID:    "sender-1",
		isGroupChat: false,
		text:        "#nuoc 150",
	})

	require.NoError(t, err)
	assert.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitPeriod, pendingRepository.created[0].ActionType)
}

func TestHandleInvoiceCommand_BatchCommand(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	// sender is manager
	userRepository.EXPECT().
		GetByZaloUserID(ctx, "sender-1").
		Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)

	houseRepository.EXPECT().
		GetHouseByCode(ctx, "manager-1", "house1").
		Return(&model.House{ID: "house-1", Name: "House 1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

	roomRepository.EXPECT().
		ListAllRoomsByHouseID(ctx, "house-1").
		Return([]model.Room{
			{ID: "room-1", Name: "P101"},
			{ID: "room-2", Name: "P102"},
		}, nil)

	houseRepository.EXPECT().
		GetByID(ctx, "house-1", "manager-1").
		Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil).AnyTimes()

	invoiceRepository.EXPECT().
		GetInvoiceByRoomAndPeriod(ctx, gomock.Any(), gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, gomock.Any(), gomock.Any()).
		Return(nil, model.ErrInvoiceNotFound).AnyTimes()

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

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
		senderID:    "sender-1",
		isGroupChat: false,
		text:        "#dien house1\np101 200",
	})

	require.NoError(t, err)
	assert.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitPeriod, pendingRepository.created[0].ActionType)
}

func TestHandleInvoiceCommand_PendingPeriodSelect(t *testing.T) {
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
			ID:         "pending-2",
			ManagerID:  "manager-1",
			ChatID:     "chat-1",
			ActionType: model.PendingActionAwaitPeriod,
			PendingData: map[string]any{
				"command_scope": "single",
				"utility_type":  "dien",
				"new_index":     int(250),
				"has_new_index": true,
				"room_id":       "room-1",
				"period_options": map[string]any{
					"5": "2026-05",
				},
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
		Return(nil, model.ErrInvoiceNotFound)
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", "2026-05").
		Return(nil, model.ErrInvoiceNotFound)

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

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
		text:     "#5",
	})

	require.NoError(t, err)
	assert.Equal(t, 250, invoiceService.input.NewElectricityIndex)
	assert.Equal(t, "2026-05", invoiceService.input.Period)
	assert.Equal(t, "pending-2", pendingRepository.deletedID)
}

func TestHandleInvoiceCommand_PendingConfirmOverwrite(t *testing.T) {
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
			ID:         "pending-3",
			ManagerID:  "manager-1",
			ChatID:     "chat-1",
			ActionType: model.PendingActionConfirmOverwrite,
			PendingData: map[string]any{
				"command_scope": "single",
				"utility_type":  "nuoc",
				"new_index":     int(300),
				"has_new_index": true,
				"period":        "2026-05",
				"room_id":       "room-1",
				"old_value":     int(250),
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
		Return(nil, model.ErrInvoiceNotFound)
	invoiceRepository.EXPECT().
		GetPreviousInvoice(ctx, "room-1", "2026-05").
		Return(nil, model.ErrInvoiceNotFound)

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

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
		text:     "#ok",
	})

	require.NoError(t, err)
	assert.Equal(t, 300, invoiceService.input.NewWaterIndex)
	assert.Equal(t, "2026-05", invoiceService.input.Period)
	assert.Equal(t, "pending-3", pendingRepository.deletedID)
}

func TestHandleInvoiceCommand_PendingCancel(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepository := mock_model.NewMockUserRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{
		pending: &model.PendingInvoiceUpdate{
			ID:         "pending-4",
			ManagerID:  "manager-1",
			ChatID:     "chat-1",
			ActionType: model.PendingActionConfirmOverwrite,
		},
	}
	zaloClient := &recordingZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	userRepository.EXPECT().
		GetByUserID(ctx, "manager-1").
		Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

	service := &zaloInvoiceCommandServiceImpl{
		userRepo:      userRepository,
		pendingRepo:   pendingRepository,
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}

	err := service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:   "chat-1",
		senderID: "sender-1",
		text:     "#huy",
	})

	require.NoError(t, err)
	assert.Equal(t, "pending-4", pendingRepository.deletedID)
	assert.Contains(t, zaloClient.messages[0], "Đã hủy")
}

func TestHasPendingState(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceWithoutRepository := &zaloInvoiceCommandServiceImpl{}
	assert.False(t, serviceWithoutRepository.HasPendingState(ctx, "manager-1", "chat-1"))

	pendingRepository := &memoryPendingInvoiceUpdateRepository{
		pending: &model.PendingInvoiceUpdate{
			ID:         "pending-1",
			ActionType: model.PendingActionConfirmOverwrite,
		},
	}
	service := &zaloInvoiceCommandServiceImpl{
		pendingRepo: pendingRepository,
	}

	assert.False(t, service.HasPendingState(ctx, "manager-1", ""))

	has := service.HasPendingState(ctx, "manager-1", "chat-1")
	assert.True(t, has)

	pendingRepository.pending = &model.PendingInvoiceUpdate{
		ActionType: model.PendingActionAwaitUtility,
	}
	has = service.HasPendingState(ctx, "manager-1", "chat-1")
	assert.False(t, has)
}
