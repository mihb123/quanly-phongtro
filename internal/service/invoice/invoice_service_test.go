package invoice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/revenue"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func ptrFloat(f float64) *float64 { return &f }
func ptrInt(i int) *int           { return &i }

func TestInvoiceServiceImpl_calculateUtilityFee(t *testing.T) {
	s := &InvoiceServiceImpl{}

	tests := []struct {
		name         string
		billingType  string
		billingUnit  string
		defaultPrice float64
		newIndex     int
		oldIndex     int
		tenantCount  int
		want         float64
		wantErr      bool
	}{
		{
			name:         "FIXED PERSON",
			billingType:  "FIXED",
			billingUnit:  "PERSON",
			defaultPrice: 100000,
			tenantCount:  2,
			want:         200000,
			wantErr:      false,
		},
		{
			name:         "FIXED ROOM",
			billingType:  "FIXED",
			billingUnit:  "ROOM",
			defaultPrice: 500000,
			tenantCount:  2,
			want:         500000,
			wantErr:      false,
		},
		{
			name:         "USAGE",
			billingType:  "USAGE",
			billingUnit:  "KWH",
			defaultPrice: 3000,
			newIndex:     100,
			oldIndex:     50,
			want:         150000,
			wantErr:      false,
		},
		{
			name:         "USAGE zero difference",
			billingType:  "USAGE",
			billingUnit:  "KWH",
			defaultPrice: 3000,
			newIndex:     50,
			oldIndex:     50,
			want:         0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.calculateUtilityFee(tt.billingType, tt.billingUnit, tt.defaultPrice, tt.newIndex, tt.oldIndex, tt.tenantCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateUtilityFee() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInvoiceService_GetInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		invoiceID string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			invoiceID: "inv-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-1").Return(&model.InvoiceWithRoom{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "Invoice not found",
			managerID: "mgr-1",
			invoiceID: "inv-not-found",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-not-found").Return(nil, model.ErrInvoiceNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			got, err := s.GetInvoice(ctx, tt.managerID, tt.invoiceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

func TestInvoiceService_ListInvoices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		filter    model.InvoiceListFilter
		setupMock func()
		wantErr   bool
		wantLen   int
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			filter:    model.InvoiceListFilter{},
			setupMock: func() {
				mockInvoiceRepo.EXPECT().ListInvoices(ctx, "mgr-1", gomock.Any()).Return([]model.InvoiceWithRoom{{}, {}}, 7, nil)
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:      "Repo error",
			managerID: "mgr-1",
			filter:    model.InvoiceListFilter{},
			setupMock: func() {
				mockInvoiceRepo.EXPECT().ListInvoices(ctx, "mgr-1", gomock.Any()).Return(nil, 0, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			got, total, err := s.ListInvoices(ctx, tt.managerID, tt.filter)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				assert.Equal(t, 7, total)
			}
		})
	}
}

func TestInvoiceService_PayInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		invoiceID string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			invoiceID: "inv-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-1").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "UNPAID"},
				}, nil)
				mockInvoiceRepo.EXPECT().UpdateInvoiceStatus(ctx, "mgr-1", "inv-1", "PAID").Return(&model.Invoice{Status: "PAID"}, nil)
			},
			wantErr: false,
		},
		{
			name:      "Already paid",
			managerID: "mgr-1",
			invoiceID: "inv-2",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-2").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "PAID"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name:      "Invoice not found",
			managerID: "mgr-1",
			invoiceID: "inv-3",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-3").Return(nil, model.ErrInvoiceNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			_, err := s.PayInvoice(ctx, tt.managerID, tt.invoiceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInvoiceService_StatusChangesPublishEvents covers event publishing for status changes.
func TestInvoiceService_StatusChangesPublishEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	eventBus := revenue.NewEventBus()
	received := make(chan interface{}, 3)
	eventBus.Subscribe(revenue.EventInvoiceChanged, func(payload interface{}) {
		received <- payload
	})
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, eventBus)
	ctx := context.Background()

	mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "pay-1").Return(&model.InvoiceWithRoom{
		Invoice: model.Invoice{Status: "UNPAID", Period: "2023-10"},
		HouseID: "house-1",
	}, nil)
	mockInvoiceRepo.EXPECT().UpdateInvoiceStatus(ctx, "mgr-1", "pay-1", "PAID").Return(&model.Invoice{Status: "PAID"}, nil)

	if _, err := s.PayInvoice(ctx, "mgr-1", "pay-1"); err != nil {
		t.Fatalf("pay invoice: %v", err)
	}

	mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "unpay-1").Return(&model.InvoiceWithRoom{
		Invoice: model.Invoice{Status: "PAID", Period: "2023-10"},
		HouseID: "house-1",
	}, nil)
	mockInvoiceRepo.EXPECT().UpdateInvoiceStatus(ctx, "mgr-1", "unpay-1", "UNPAID").Return(&model.Invoice{Status: "UNPAID"}, nil)

	if _, err := s.UnpayInvoice(ctx, "mgr-1", "unpay-1"); err != nil {
		t.Fatalf("unpay invoice: %v", err)
	}

	mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "delete-1").Return(&model.InvoiceWithRoom{
		Invoice: model.Invoice{Status: "UNPAID", Period: "2023-10"},
		HouseID: "house-1",
	}, nil)
	mockInvoiceRepo.EXPECT().DeleteInvoice(ctx, "mgr-1", "delete-1").Return(nil)

	if err := s.DeleteInvoice(ctx, "mgr-1", "delete-1"); err != nil {
		t.Fatalf("delete invoice: %v", err)
	}

	for i := 0; i < 3; i++ {
		select {
		case value := <-received:
			payload, ok := value.(revenue.RevenueSummaryPayload)
			if !ok || payload.HouseID != "house-1" || payload.Period != "2023-10" {
				t.Fatalf("unexpected payload: %+v", value)
			}
		case <-time.After(time.Second):
			t.Fatal("expected invoice changed event")
		}
	}
}

func TestInvoiceService_DeleteInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		invoiceID string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			invoiceID: "inv-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-1").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "UNPAID"},
				}, nil)
				mockInvoiceRepo.EXPECT().DeleteInvoice(ctx, "mgr-1", "inv-1").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "Cannot delete paid",
			managerID: "mgr-1",
			invoiceID: "inv-2",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-2").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "PAID"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name:      "Invoice not found",
			managerID: "mgr-1",
			invoiceID: "inv-3",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-3").Return(nil, model.ErrInvoiceNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := s.DeleteInvoice(ctx, tt.managerID, tt.invoiceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvoiceService_CreateInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)

	s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		input     CreateInvoiceInput
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path - Create new invoice",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 100,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil)
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "Create invoice repo error",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 100,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:      "Update existing unpaid invoice",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 100,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(&model.Invoice{ID: "inv-1", Status: "UNPAID"}, nil)
				mockInvoiceRepo.EXPECT().UpdateInvoice(ctx, "mgr-1", gomock.Any()).Return(nil)
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "Update existing unpaid invoice error",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 100,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(&model.Invoice{ID: "inv-1", Status: "UNPAID"}, nil)
				mockInvoiceRepo.EXPECT().UpdateInvoice(ctx, "mgr-1", gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:      "Cannot edit a paid invoice",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 100,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(&model.Invoice{ID: "inv-1", Status: "PAID"}, nil)
			},
			wantErr: true,
		},
		{
			name:      "Room not found (not owned)",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID: "room-1",
				Period: "01/2023",
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(false, nil)
			},
			wantErr: true,
		},
		{
			name:      "Invalid electricity index (less than old index)",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "01/2023",
				NewElectricityIndex: 50,
				NewWaterIndex:       10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)

				// Previous invoice had index 60
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(&model.Invoice{NewElectricityIndex: 60, NewWaterIndex: 5}, nil)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)
			},
			wantErr: true,
		},
		{
			name:      "GetRoomByIDOnly fails",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name:      "IsHouseOwnedBy fails",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(false, errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name:      "GetByID (house) fails",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name:      "GetPreviousInvoice returns non-ErrInvoiceNotFound",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1"}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name:      "GetCurrentNumTenantInRoom fails",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1"}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(0), errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name:      "Room has custom prices & Extra fees",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID: "room-1", Period: "01/2023", VehicleCount: 2, OtherFee: 50, Discount: 10,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{
					ID: "room-1", HouseID: "house-1", Price: 1000,
					ElectricityPrice: ptrFloat(4000), WaterPrice: ptrFloat(20000),
					WifiPrice: ptrFloat(50000), ServicePrice: ptrFloat(30000),
					ParkingPrice: ptrFloat(150000), ExtraPersonFee: ptrFloat(100000),
					MaxTenants: 2, ExtraPersonThreshold: ptrInt(2), ExtraVehicleThreshold: ptrInt(1),
					ExtraVehicleFee: ptrFloat(50000),
				}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1",
					ElectricityBillingType: "FIXED", WaterBillingType: "FIXED",
					ElectricityBillingUnit: "PERSON", WaterBillingUnit: "PERSON",
				}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(3), nil) // 3 tenants > max 2 => extra fee
				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil)
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "FIXED billing type with index normalization & invalid water index",
			managerID: "mgr-1",
			input: CreateInvoiceInput{
				RoomID: "room-1", Period: "01/2023", NewWaterIndex: 5,
			},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1",
					ElectricityBillingType: "FIXED", WaterBillingType: "USAGE",
				}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(&model.Invoice{NewElectricityIndex: 0, NewWaterIndex: 10}, nil)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(1), nil)
			},
			wantErr: true,
		},
		{
			name:      "GetInvoiceByRoomAndPeriod returns unexpected error",
			managerID: "mgr-1",
			input:     CreateInvoiceInput{RoomID: "room-1", Period: "01/2023"},
			setupMock: func() {
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)
				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			_, err := s.CreateInvoice(ctx, tt.managerID, tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvoiceService_CreateInvoiceRoomFeeForVacantRoom(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }
	tenantCount := 0

	tests := []struct {
		name        string
		roomStatus  string
		exclude     *bool
		wantRoomFee float64
	}{
		{name: "Vacant room skips room fee by default", roomStatus: "AVAILABLE", wantRoomFee: 0},
		{name: "Occupied room charges room fee by default", roomStatus: "OCCUPIED", wantRoomFee: 1000},
		{name: "Explicit exclude on occupied room", roomStatus: "OCCUPIED", exclude: boolPtr(true), wantRoomFee: 0},
		{name: "Explicit include on vacant room", roomStatus: "AVAILABLE", exclude: boolPtr(false), wantRoomFee: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)
			s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, nil)
			ctx := context.Background()

			mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000, Status: tt.roomStatus}, nil)
			mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
			mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE", DefaultElectricityPrice: 3000}, nil)
			mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-10").Return(&model.Invoice{NewElectricityIndex: 100, NewWaterIndex: 10}, nil)
			mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-10").Return(nil, model.ErrInvoiceNotFound)

			var saved *model.Invoice
			mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, inv *model.Invoice) error {
				saved = inv
				return nil
			})
			mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)

			_, err := s.CreateInvoice(ctx, "mgr-1", CreateInvoiceInput{
				RoomID:              "room-1",
				Period:              "2026-10",
				NewElectricityIndex: 120,
				NewWaterIndex:       10,
				TenantCount:         &tenantCount,
				ExcludeRoomFee:      tt.exclude,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantRoomFee, saved.RoomFee)
			require.Equal(t, 60000.0, saved.ElectricityFee)
			require.Equal(t, tt.wantRoomFee+60000, saved.TotalAmount)
		})
	}
}

func TestInvoiceService_CreateInvoiceOtherFeeItems(t *testing.T) {
	tenantCount := 1
	tests := []struct {
		name          string
		otherFee      float64
		otherFees     []model.InvoiceFeeItem
		wantOtherFee  float64
		wantOtherFees []model.InvoiceFeeItem
	}{
		{
			name:          "Items replace the legacy amount",
			otherFee:      999,
			otherFees:     []model.InvoiceFeeItem{{Name: " Vệ sinh trả phòng ", Amount: 200000}, {Name: "", Amount: 50000}, {Name: "Bỏ qua", Amount: 0}},
			wantOtherFee:  250000,
			wantOtherFees: []model.InvoiceFeeItem{{Name: "Vệ sinh trả phòng", Amount: 200000}, {Name: "Chi phí phát sinh", Amount: 50000}},
		},
		{
			name:          "Empty list clears other fees",
			otherFee:      999,
			otherFees:     []model.InvoiceFeeItem{},
			wantOtherFee:  0,
			wantOtherFees: []model.InvoiceFeeItem{},
		},
		{
			name:          "Missing list keeps legacy amount",
			otherFee:      30000,
			wantOtherFee:  30000,
			wantOtherFees: []model.InvoiceFeeItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mock_model.NewMockTenantRepository(ctrl), nil)
			ctx := context.Background()

			mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000, Status: "OCCUPIED"}, nil)
			mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
			mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
			mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "2026-10").Return(nil, model.ErrInvoiceNotFound)
			mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "2026-10").Return(nil, model.ErrInvoiceNotFound)

			var saved *model.Invoice
			mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, inv *model.Invoice) error {
				saved = inv
				return nil
			})
			mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)

			_, err := s.CreateInvoice(ctx, "mgr-1", CreateInvoiceInput{
				RoomID:      "room-1",
				Period:      "2026-10",
				TenantCount: &tenantCount,
				OtherFee:    tt.otherFee,
				OtherFees:   tt.otherFees,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantOtherFee, saved.OtherFee)
			require.Equal(t, tt.wantOtherFees, saved.OtherFees)
			require.Equal(t, 1000+tt.wantOtherFee, saved.TotalAmount)
		})
	}
}

func TestInvoiceService_UnpayInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		invoiceID string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			invoiceID: "inv-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-1").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "PAID"},
				}, nil)
				mockInvoiceRepo.EXPECT().UpdateInvoiceStatus(ctx, "mgr-1", "inv-1", "UNPAID").Return(&model.Invoice{Status: "UNPAID"}, nil)
			},
			wantErr: false,
		},
		{
			name:      "Already unpaid",
			managerID: "mgr-1",
			invoiceID: "inv-2",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-2").Return(&model.InvoiceWithRoom{
					Invoice: model.Invoice{Status: "UNPAID"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name:      "Invoice not found",
			managerID: "mgr-1",
			invoiceID: "inv-3",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", "inv-3").Return(nil, model.ErrInvoiceNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			_, err := s.UnpayInvoice(ctx, tt.managerID, tt.invoiceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInvoiceService_CreateInvoicePublishesEvent covers create-invoice recalculation events.
func TestInvoiceService_CreateInvoicePublishesEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)
	eventBus := revenue.NewEventBus()
	received := make(chan interface{}, 1)
	eventBus.Subscribe(revenue.EventInvoiceChanged, func(payload interface{}) {
		received <- payload
	})

	s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, eventBus)
	ctx := context.Background()
	input := CreateInvoiceInput{
		RoomID:              "room-1",
		Period:              "01/2023",
		NewElectricityIndex: 100,
		NewWaterIndex:       10,
	}

	mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
	mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
	mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
	mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
	mockTenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "room-1").Return(int64(2), nil)
	mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
	mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil)
	mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)

	if _, err := s.CreateInvoice(ctx, "mgr-1", input); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	select {
	case value := <-received:
		payload, ok := value.(revenue.RevenueSummaryPayload)
		if !ok || payload.HouseID != "house-1" || payload.Period != "01/2023" {
			t.Fatalf("unexpected payload: %+v", value)
		}
	case <-time.After(time.Second):
		t.Fatal("expected invoice changed event")
	}
}

func TestInvoiceService_RecalculateUnpaidInvoicesByRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)

	s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		roomID    string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			roomID:    "room-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return([]model.Invoice{
					{
						RoomID: "room-1", Period: "01/2023", OldElectricityIndex: 10, NewElectricityIndex: 20,
						OldWaterIndex: 5, NewWaterIndex: 10, OtherFee: 0, Discount: 0, VehicleCount: 1, TenantCount: 2,
					},
				}, nil)

				// Inside CreateInvoice calls
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1000}, nil)
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil)
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil)
				mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)

				mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", "01/2023").Return(nil, model.ErrInvoiceNotFound)
				mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil)
				mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "GetUnpaidInvoicesByRoomID error",
			managerID: "mgr-1",
			roomID:    "room-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:      "CreateInvoice error is logged and ignored",
			managerID: "mgr-1",
			roomID:    "room-1",
			setupMock: func() {
				mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return([]model.Invoice{
					{
						RoomID: "room-1", Period: "01/2023", OldElectricityIndex: 10, NewElectricityIndex: 20,
						OldWaterIndex: 5, NewWaterIndex: 10, OtherFee: 0, Discount: 0, VehicleCount: 1, TenantCount: 2,
					},
				}, nil)
				mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(nil, errors.New("room repo error"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := s.RecalculateUnpaidInvoicesByRoom(ctx, tt.managerID, tt.roomID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvoiceService_RecalculateKeepsExcludedRoomFee(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)
	s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, nil)
	ctx := context.Background()

	mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return([]model.Invoice{
		{RoomID: "room-1", Period: "2026-09", RoomFee: 0, NewElectricityIndex: 20, NewWaterIndex: 10},
		{RoomID: "room-1", Period: "2026-10", RoomFee: 1000, NewElectricityIndex: 30, NewWaterIndex: 12},
	}, nil)
	mockRoomRepo.EXPECT().GetRoomByIDOnly(ctx, "room-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Price: 1500, Status: "AVAILABLE"}, nil).Times(2)
	mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "mgr-1").Return(true, nil).Times(2)
	mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "mgr-1").Return(&model.House{ID: "house-1", ElectricityBillingType: "USAGE", WaterBillingType: "USAGE"}, nil).Times(2)
	mockInvoiceRepo.EXPECT().GetPreviousInvoice(ctx, "room-1", gomock.Any()).Return(nil, model.ErrInvoiceNotFound).Times(2)
	mockInvoiceRepo.EXPECT().GetInvoiceByRoomAndPeriod(ctx, "room-1", gomock.Any()).Return(nil, model.ErrInvoiceNotFound).Times(2)

	roomFees := map[string]float64{}
	mockInvoiceRepo.EXPECT().CreateInvoice(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, inv *model.Invoice) error {
		roomFees[inv.Period] = inv.RoomFee
		return nil
	}).Times(2)
	mockInvoiceRepo.EXPECT().GetInvoiceByID(ctx, "mgr-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil).Times(2)

	require.NoError(t, s.RecalculateUnpaidInvoicesByRoom(ctx, "mgr-1", "room-1"))
	require.Equal(t, map[string]float64{"2026-09": 0, "2026-10": 1500}, roomFees)
}

func TestInvoiceService_RecalculateUnpaidInvoicesByHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockTenantRepo := mock_model.NewMockTenantRepository(ctrl)

	s := NewInvoiceService(mockInvoiceRepo, mockRoomRepo, mockHouseRepo, mockTenantRepo, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID string
		houseID   string
		setupMock func()
		wantErr   bool
	}{
		{
			name:      "Happy path",
			managerID: "mgr-1",
			houseID:   "house-1",
			setupMock: func() {
				mockRoomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{
					{ID: "room-1"},
				}, nil)

				// RecalculateUnpaidInvoicesByRoom calls
				mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return([]model.Invoice{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "ListAllRoomsByHouseID error",
			managerID: "mgr-1",
			houseID:   "house-1",
			setupMock: func() {
				mockRoomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:      "Room recalculation error is logged and ignored",
			managerID: "mgr-1",
			houseID:   "house-1",
			setupMock: func() {
				mockRoomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1"}}, nil)
				mockInvoiceRepo.EXPECT().GetUnpaidInvoicesByRoomID(ctx, "room-1").Return(nil, errors.New("db error"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := s.RecalculateUnpaidInvoicesByHouse(ctx, tt.managerID, tt.houseID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInvoiceServiceResolveTransactionImagePath verifies image ownership and traversal checks.
func TestInvoiceServiceResolveTransactionImagePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockInvoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	invoiceService := NewInvoiceService(mockInvoiceRepo, nil, nil, nil, nil)

	mockInvoiceRepo.EXPECT().
		GetInvoiceByTransactionImagePath(ctx, "mgr-1", "/uploads/transactions/tx.jpg").
		Return(&model.InvoiceWithRoom{Invoice: model.Invoice{ID: "inv-1"}}, nil)

	filePath, err := invoiceService.ResolveTransactionImagePath(ctx, "mgr-1", "tx.jpg")
	require.NoError(t, err)
	require.Equal(t, "uploads/transactions/tx.jpg", filePath)

	_, err = invoiceService.ResolveTransactionImagePath(ctx, "mgr-1", "../secret.jpg")
	require.ErrorIs(t, err, shared.ErrInvalidInput)
}
