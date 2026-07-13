package zalo

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type errorInvoiceService struct {
	invoicesvc.InvoiceService
	err error
}

// CreateInvoice returns the configured error for retryable invoice command tests.
func (s *errorInvoiceService) CreateInvoice(context.Context, string, invoicesvc.CreateInvoiceInput) (*model.InvoiceWithRoom, error) {
	return nil, s.err
}

// TestApplyUtilityUpdateRequiresReading verifies usage utilities require a new meter index.
func TestApplyUtilityUpdateRequiresReading(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE"}, nil)
	service := &zaloInvoiceCommandServiceImpl{houseRepo: houseRepository}

	_, pendingCreated, err := service.applyUtilityUpdate(ctx, "manager-1", webhookMessageContext{}, utilityUpdateRequest{
		UtilityType:  "dien",
		Room:         model.Room{ID: "room-1", HouseID: "house-1"},
		ForcedPeriod: "2026-05",
	})

	require.Error(t, err)
	assert.False(t, pendingCreated)
	assert.Contains(t, err.Error(), "chỉ số")
}

// TestApplyUtilityUpdateReturnsPreviousInvoiceError verifies build errors bubble up.
func TestApplyUtilityUpdateReturnsPreviousInvoiceError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE"}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)
	dbErr := errors.New("previous lookup failed")
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(nil, dbErr)

	service := &zaloInvoiceCommandServiceImpl{houseRepo: houseRepository, invoiceRepo: invoiceRepository}
	_, pendingCreated, err := service.applyUtilityUpdate(ctx, "manager-1", webhookMessageContext{}, utilityUpdateRequest{
		UtilityType:  "dien",
		Room:         model.Room{ID: "room-1", HouseID: "house-1"},
		NewIndex:     200,
		HasNewIndex:  true,
		ForcedPeriod: "2026-05",
	})

	require.ErrorIs(t, err, dbErr)
	assert.False(t, pendingCreated)
}

// TestApplyUtilityUpdateCreatesRetryPendingOnInvalidIndex verifies retryable index errors store pending state.
func TestApplyUtilityUpdateCreatesRetryPendingOnInvalidIndex(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE"}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: &errorInvoiceService{err: model.ErrInvalidElectricityIndex},
		invoiceRepo:    invoiceRepository,
		houseRepo:      houseRepository,
		userRepo:       userRepository,
		pendingRepo:    pendingRepository,
		zaloClient:     zaloClient,
		encryptionKey:  encryptionKey,
	}
	_, pendingCreated, err := service.applyUtilityUpdate(ctx, "manager-1", webhookMessageContext{chatID: "chat-1"}, utilityUpdateRequest{
		UtilityType:  "dien",
		Room:         model.Room{ID: "room-1", HouseID: "house-1"},
		NewIndex:     90,
		HasNewIndex:  true,
		ForcedPeriod: "2026-05",
	})

	require.NoError(t, err)
	assert.True(t, pendingCreated)
	require.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionAwaitUtility, pendingRepository.created[0].ActionType)
	assert.Contains(t, zaloClient.messages[0], "#dien")
}

// TestApplyUtilityUpdateCreatesOverwritePending verifies existing readings require confirmation.
func TestApplyUtilityUpdateCreatesOverwritePending(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	zaloClient := &recordingZaloClient{}

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE"}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(&model.Invoice{NewElectricityIndex: 100}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceRepo:   invoiceRepository,
		houseRepo:     houseRepository,
		userRepo:      userRepository,
		pendingRepo:   pendingRepository,
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}
	_, pendingCreated, err := service.applyUtilityUpdate(ctx, "manager-1", webhookMessageContext{chatID: "chat-1"}, utilityUpdateRequest{
		UtilityType:  "dien",
		Room:         model.Room{ID: "room-1", HouseID: "house-1"},
		NewIndex:     200,
		HasNewIndex:  true,
		ForcedPeriod: "2026-05",
	})

	require.NoError(t, err)
	assert.True(t, pendingCreated)
	require.Len(t, pendingRepository.created, 1)
	assert.Equal(t, model.PendingActionConfirmOverwrite, pendingRepository.created[0].ActionType)
	assert.Contains(t, zaloClient.messages[0], "#ok")
}

// TestApplyUtilityUpdateCompleteResult verifies successful updates return a complete command result.
func TestApplyUtilityUpdateCompleteResult(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseRepository := mock_model.NewMockHouseRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	houseRepository.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "FIXED"}, nil)
	invoiceRepository.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-05").Return(nil, model.ErrInvoiceNotFound)
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(&model.Invoice{NewElectricityIndex: 100, NewWaterIndex: 20}, nil)

	service := &zaloInvoiceCommandServiceImpl{
		invoiceService: &recordingInvoiceService{},
		invoiceRepo:    invoiceRepository,
		houseRepo:      houseRepository,
	}
	result, pendingCreated, err := service.applyUtilityUpdate(ctx, "manager-1", webhookMessageContext{}, utilityUpdateRequest{
		UtilityType:  "dien",
		Room:         model.Room{ID: "room-1", HouseID: "house-1", Name: "P101"},
		NewIndex:     150,
		HasNewIndex:  true,
		ForcedPeriod: "2026-05",
	})

	require.NoError(t, err)
	assert.False(t, pendingCreated)
	assert.True(t, result.Complete)
	assert.Equal(t, 100, result.OldIndex)
	assert.Equal(t, 150, result.NewIndex)
}

// TestBuildInvoiceInputPreservesExistingInvoiceValues verifies overwrite updates keep unrelated fields.
func TestBuildInvoiceInputPreservesExistingInvoiceValues(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	invoiceRepository.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-05").Return(&model.Invoice{NewElectricityIndex: 100, NewWaterIndex: 20}, nil)
	service := &zaloInvoiceCommandServiceImpl{invoiceRepo: invoiceRepository}

	input, oldIndex, err := service.buildInvoiceInput(ctx, utilityUpdateRequest{
		UtilityType: "nuoc",
		Room:        model.Room{ID: "room-1"},
		NewIndex:    45,
		HasNewIndex: true,
	}, "2026-05", &model.Invoice{
		OldElectricityIndex: 80,
		NewElectricityIndex: 120,
		OldWaterIndex:       25,
		NewWaterIndex:       30,
		OtherFee:            10000,
		Discount:            5000,
		VehicleCount:        2,
		TenantCount:         3,
	})

	require.NoError(t, err)
	assert.Equal(t, 25, oldIndex)
	assert.Equal(t, 120, input.NewElectricityIndex)
	assert.Equal(t, 45, input.NewWaterIndex)
	assert.Equal(t, 10000.0, input.OtherFee)
	assert.Equal(t, 5000.0, input.Discount)
	require.NotNil(t, input.TenantCount)
	assert.Equal(t, 3, *input.TenantCount)
}
