package zalo

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakePaymentService lets delivery tests control preferred payment link results.
type fakePaymentService struct {
	paymentsvc.PaymentService
	link *model.InvoicePaymentLink
	err  error
}

type flowZaloClient struct {
	ZaloClient
	sendMessageErr error
	sendPhotoErr   error
	messages       []string
	photos         []string
}

// SendMessage records outgoing command messages and returns the configured error.
func (c *flowZaloClient) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	c.messages = append(c.messages, text)
	return c.sendMessageErr
}

// SendPhoto records outgoing invoice photos and returns the configured error.
func (c *flowZaloClient) SendPhoto(ctx context.Context, botToken, chatID, photoURL, caption string) error {
	c.photos = append(c.photos, photoURL)
	return c.sendPhotoErr
}

type flowImageService struct {
	image []byte
	err   error
}

// GenerateInvoiceImage returns the configured invoice image bytes for delivery flows.
func (s *flowImageService) GenerateInvoiceImage(ctx context.Context, invoice *model.InvoiceWithRoom) ([]byte, error) {
	return s.image, s.err
}

// CreatePreferredPaymentLinkForInvoice returns the pre-configured link and error.
func (f *fakePaymentService) CreatePreferredPaymentLinkForInvoice(ctx context.Context, managerID string, invoice *model.InvoiceWithRoom, tenantName string) (*model.InvoicePaymentLink, error) {
	return f.link, f.err
}

// TestNewZaloInvoiceCommandService verifies the constructor wires the concrete implementation.
func TestNewZaloInvoiceCommandService(t *testing.T) {
	service := NewZaloInvoiceCommandService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, []byte("key"), "https://example.com/")
	impl, ok := service.(*zaloInvoiceCommandServiceImpl)
	require.True(t, ok)
	assert.Equal(t, "https://example.com", impl.publicBaseURL)
}

// TestSetInvoiceCommandService verifies the optional command handler is attached.
func TestSetInvoiceCommandService(t *testing.T) {
	svc, err := NewZaloService(nil, nil, nil, nil, nil, nil, nil, nil, "z123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	impl := svc.(*zaloServiceImpl)
	commandService := &zaloInvoiceCommandServiceImpl{}
	impl.SetInvoiceCommandService(commandService)
	assert.Equal(t, ZaloInvoiceCommandService(commandService), impl.invoiceCommandService)
}

// TestSendTextMessage_AuthErrorMarksInactive covers the invalid-token deactivation branch.
func TestSendTextMessage_AuthErrorMarksInactive(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepository := mock_model.NewMockUserRepository(ctrl)
	zaloClient := &flowZaloClient{sendMessageErr: errors.New("invalid access token -216")}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil)
	userRepository.EXPECT().UpdateUser(ctx, "manager-1", gomock.Any()).Return(&model.User{}, nil)

	service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, zaloClient: zaloClient, encryptionKey: encryptionKey}
	err := service.sendTextMessage(ctx, "manager-1", "chat-1", "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

// TestSendTextMessage_TokenErrors covers missing config and decrypt failures.
func TestSendTextMessage_TokenErrors(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")

	t.Run("no token configured", func(t *testing.T) {
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, encryptionKey: encryptionKey}
		err := service.sendTextMessage(ctx, "manager-1", "chat-1", "hi")
		require.Error(t, err)
	})

	t.Run("decrypt fails", func(t *testing.T) {
		userRepository := mock_model.NewMockUserRepository(ctrl)
		bad := "not-encrypted"
		userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &bad}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, encryptionKey: encryptionKey}
		err := service.sendTextMessage(ctx, "manager-1", "chat-1", "hi")
		require.Error(t, err)
	})
}

// TestHandleSingleCommand_RoomResolutionError verifies unresolved rooms send an error reply.
func TestHandleSingleCommand_RoomResolutionError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepository := mock_model.NewMockUserRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	zaloClient := &flowZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	roomRepository.EXPECT().GetRoomByGroupChatID(ctx, "chat-group").Return(nil, errors.New("not found"))
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil)

	service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, roomRepo: roomRepository, zaloClient: zaloClient, encryptionKey: encryptionKey}
	parsed := &ParsedCommand{Type: CommandUtilitySingle, UtilityType: "dien"}
	err := service.handleSingleCommand(ctx, "manager-1", webhookMessageContext{chatID: "chat-group", isGroupChat: true}, parsed, "", false)
	require.NoError(t, err)
}

// TestResolveSingleCommandRoom covers group, manager, and tenant resolution branches.
func TestResolveSingleCommandRoom(t *testing.T) {
	ctx := context.Background()

	t.Run("group not linked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		roomRepository := mock_model.NewMockRoomRepository(ctrl)
		roomRepository.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(nil, errors.New("nf"))
		service := &zaloInvoiceCommandServiceImpl{roomRepo: roomRepository}
		_, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{chatID: "g1", isGroupChat: true})
		require.Error(t, err)
	})

	t.Run("group not owned by manager", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		roomRepository := mock_model.NewMockRoomRepository(ctrl)
		houseRepository := mock_model.NewMockHouseRepository(ctrl)
		roomRepository.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
		houseRepository.EXPECT().GetByID(ctx, "h1", "manager-1").Return(nil, errors.New("nf"))
		service := &zaloInvoiceCommandServiceImpl{roomRepo: roomRepository, houseRepo: houseRepository}
		_, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{chatID: "g1", isGroupChat: true})
		require.Error(t, err)
	})

	t.Run("private account not linked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(nil, errors.New("nf"))
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository}
		_, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender"})
		require.Error(t, err)
	})

	t.Run("private manager rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository}
		_, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender"})
		require.Error(t, err)
	})

	t.Run("private tenant not renting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		tenantRepository := mock_model.NewMockTenantRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)
		tenantRepository.EXPECT().GetFirstTenantByUserID(ctx, "manager-1", "user-1").Return(nil, errors.New("nf"))
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, tenantRepo: tenantRepository}
		_, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender"})
		require.Error(t, err)
	})

	t.Run("private tenant resolves room", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		tenantRepository := mock_model.NewMockTenantRepository(ctrl)
		roomRepository := mock_model.NewMockRoomRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)
		tenantRepository.EXPECT().GetFirstTenantByUserID(ctx, "manager-1", "user-1").Return(&model.FullInfoTenant{RoomID: "r1"}, nil)
		roomRepository.EXPECT().GetRoomByIDOnly(ctx, "r1").Return(&model.Room{ID: "r1"}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, tenantRepo: tenantRepository, roomRepo: roomRepository}
		room, err := service.resolveSingleCommandRoom(ctx, "manager-1", webhookMessageContext{senderID: "sender"})
		require.NoError(t, err)
		assert.Equal(t, "r1", room.ID)
	})
}

// TestEnsureManagerSender covers link, ownership, and role verification.
func TestEnsureManagerSender(t *testing.T) {
	ctx := context.Background()

	t.Run("not linked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(nil, errors.New("nf"))
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository}
		require.Error(t, service.ensureManagerSender(ctx, "manager-1", "sender"))
	})

	t.Run("different manager", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "other", Role: model.RoleManager}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository}
		require.Error(t, service.ensureManagerSender(ctx, "manager-1", "sender"))
	})

	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepository := mock_model.NewMockUserRepository(ctrl)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository}
		require.NoError(t, service.ensureManagerSender(ctx, "manager-1", "sender"))
	})
}

// TestHandleBatchCommand_EarlyValidation covers the pre-flight guard branches.
func TestHandleBatchCommand_EarlyValidation(t *testing.T) {
	ctx := context.Background()
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")

	newService := func(t *testing.T) (*zaloInvoiceCommandServiceImpl, *mock_model.MockUserRepository, *mock_model.MockHouseRepository, *flowZaloClient) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		userRepository := mock_model.NewMockUserRepository(ctrl)
		houseRepository := mock_model.NewMockHouseRepository(ctrl)
		zaloClient := &flowZaloClient{}
		token := encryptedToken(t, encryptionKey)
		userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
		service := &zaloInvoiceCommandServiceImpl{userRepo: userRepository, houseRepo: houseRepository, zaloClient: zaloClient, encryptionKey: encryptionKey}
		return service, userRepository, houseRepository, zaloClient
	}

	t.Run("group chat rejected", func(t *testing.T) {
		service, _, _, _ := newService(t)
		parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1"}
		require.NoError(t, service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "chat-1", isGroupChat: true}, parsed, "", false))
	})

	t.Run("sender not manager", func(t *testing.T) {
		service, userRepository, _, _ := newService(t)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(nil, errors.New("nf"))
		parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1"}
		require.NoError(t, service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "", false))
	})

	t.Run("missing house code", func(t *testing.T) {
		service, userRepository, _, _ := newService(t)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
		parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien"}
		require.NoError(t, service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "", false))
	})

	t.Run("no entries", func(t *testing.T) {
		service, userRepository, _, _ := newService(t)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
		parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1"}
		require.NoError(t, service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "", false))
	})

	t.Run("house not found", func(t *testing.T) {
		service, userRepository, houseRepository, _ := newService(t)
		userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
		houseRepository.EXPECT().GetHouseByCode(ctx, "manager-1", "h1").Return(nil, errors.New("nf"))
		parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1", Entries: []RoomUtilityEntry{{RoomName: "P101", NewIndex: 100, HasNewIndex: true}}}
		require.NoError(t, service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "", false))
	})
}

// TestHandleBatchCommand_OverwriteConfirmation covers the batch overwrite pending flow.
func TestHandleBatchCommand_OverwriteConfirmation(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepository := mock_model.NewMockUserRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	zaloClient := &flowZaloClient{}
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().GetHouseByCode(ctx, "manager-1", "h1").Return(&model.House{ID: "house-1", Name: "House 1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
	roomRepository.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", Name: "P101"}}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(&model.Invoice{
		ID: "existing", RoomID: "room-1", Period: "2026-05", OldElectricityIndex: 100, NewElectricityIndex: 150,
	}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		userRepo:      userRepository,
		houseRepo:     houseRepository,
		roomRepo:      roomRepository,
		invoiceRepo:   invoiceRepository,
		pendingRepo:   pendingRepository,
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}
	parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1", Entries: []RoomUtilityEntry{{RoomName: "P101", NewIndex: 200, HasNewIndex: true}}}
	err := service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "2026-05", false)
	require.NoError(t, err)
	require.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionConfirmOverwrite, pendingRepository.created[0].ActionType)
}

// TestHandleBatchCommand_CompleteDelivers covers batch apply, delivery, and summary reporting.
func TestHandleBatchCommand_CompleteDelivers(t *testing.T) {
	t.Cleanup(func() { _ = os.RemoveAll("uploads/zalo-invoices") })
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	userRepository := mock_model.NewMockUserRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	imageService := &flowImageService{image: []byte("img")}
	zaloClient := &flowZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	// water FIXED so only electricity reading is required and invoice completes.
	house := &model.House{ID: "house-1", Name: "House 1", ElectricityBillingType: "USAGE", WaterBillingType: "FIXED"}

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().GetHouseByCode(ctx, "manager-1", "h1").Return(house, nil)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(house, nil).AnyTimes()
	roomRepository.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", Name: "P101", HouseID: "house-1"}}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)

	// delivery
	deliverInvoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", Period: "2026-05", TotalAmount: 1000}, RoomName: "P101", HouseID: "house-1"}
	invoiceRepository.EXPECT().GetInvoiceByID(ctx, "manager-1", "invoice-1").Return(deliverInvoice, nil)
	roomRepository.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(&model.Room{ID: "room-1", GroupChatID: ptrString("group-1")}, nil)
	tenantRepository.EXPECT().ListTenantByRoomID(ctx, "manager-1", "room-1").Return([]model.FullInfoTenant{}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepository,
		roomRepo:       roomRepository,
		houseRepo:      houseRepository,
		tenantRepo:     tenantRepository,
		userRepo:       userRepository,
		zaloClient:     zaloClient,
		imageService:   imageService,
		encryptionKey:  encryptionKey,
		publicBaseURL:  "https://example.com",
	}
	parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1", Entries: []RoomUtilityEntry{
		{RoomName: "P101", NewIndex: 200, HasNewIndex: true},
		{RoomName: "P999", NewIndex: 300, HasNewIndex: true}, // unmatched -> failed line
	}}
	err := service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "2026-05", true)
	require.NoError(t, err)
}

// TestHandleBatchCommand_MissingUtility covers the incomplete batch summary branch.
func TestHandleBatchCommand_MissingUtility(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceService := &recordingInvoiceService{}
	userRepository := mock_model.NewMockUserRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &flowZaloClient{}
	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	// both utilities USAGE, only electricity supplied -> water missing -> incomplete.
	house := &model.House{ID: "house-1", Name: "House 1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}

	userRepository.EXPECT().GetByZaloUserID(ctx, "sender").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().GetHouseByCode(ctx, "manager-1", "h1").Return(house, nil)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(house, nil).AnyTimes()
	roomRepository.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", Name: "P101", HouseID: "house-1"}}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound).AnyTimes()
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)

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
	parsed := &ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "h1", Entries: []RoomUtilityEntry{{RoomName: "P101", NewIndex: 200, HasNewIndex: true}}}
	err := service.handleBatchCommand(ctx, "manager-1", webhookMessageContext{chatID: "sender", senderID: "sender"}, parsed, "2026-05", true)
	require.NoError(t, err)
	// incomplete batch result stores an await-utility pending state.
	require.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitUtility, pendingRepository.created[0].ActionType)
}

// ptrString returns a pointer to a string literal for optional model fields.
func ptrString(v string) *string { return &v }
