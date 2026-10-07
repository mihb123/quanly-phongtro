package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service/revenue"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"
)

type CreateInvoiceInput struct {
	RoomID              string
	Period              string
	OldElectricityIndex *int
	NewElectricityIndex int
	OldWaterIndex       *int
	NewWaterIndex       int
	OtherFee            float64
	OtherFees           []model.InvoiceFeeItem
	Discount            float64
	VehicleCount        int
	TenantCount         *int
	ExcludeRoomFee      *bool
}

type InvoiceService interface {
	CreateInvoice(ctx context.Context, managerID string, input CreateInvoiceInput) (*model.InvoiceWithRoom, error)
	GetInvoice(ctx context.Context, managerID, invoiceID string) (*model.InvoiceWithRoom, error)
	ListInvoices(ctx context.Context, managerID string, filter model.InvoiceListFilter) ([]model.InvoiceWithRoom, int, error)
	SumInvoicesByStatus(ctx context.Context, managerID string, filter model.InvoiceBreakdownFilter) ([]model.InvoiceStatusTotals, error)
	PayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error)
	UnpayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error)
	RecalculateUnpaidInvoicesByRoom(ctx context.Context, managerID, roomID string) error
	RecalculateUnpaidInvoicesByHouse(ctx context.Context, managerID, houseID string) error
	DeleteInvoice(ctx context.Context, managerID, invoiceID string) error
	ResolveTransactionImagePath(ctx context.Context, managerID, requestPath string) (string, error)
}

type InvoiceServiceImpl struct {
	invoiceRepo model.InvoiceRepository
	roomRepo    model.RoomRepository
	houseRepo   model.HouseRepository
	tenantRepo  model.TenantRepository
	eventBus    revenue.EventBus
}

func NewInvoiceService(invoiceRepo model.InvoiceRepository, roomRepo model.RoomRepository, houseRepo model.HouseRepository, tenantRepo model.TenantRepository, eventBus revenue.EventBus) InvoiceService {
	return &InvoiceServiceImpl{
		invoiceRepo: invoiceRepo,
		roomRepo:    roomRepo,
		houseRepo:   houseRepo,
		tenantRepo:  tenantRepo,
		eventBus:    eventBus,
	}
}

func (s *InvoiceServiceImpl) CreateInvoice(ctx context.Context, managerID string, input CreateInvoiceInput) (*model.InvoiceWithRoom, error) {
	room, err := s.roomRepo.GetRoomByIDOnly(ctx, input.RoomID)
	if err != nil {
		return nil, fmt.Errorf("get room for invoice: %w", err)
	}

	owned, err := s.houseRepo.IsHouseOwnedBy(ctx, room.HouseID, managerID)
	if err != nil {
		return nil, fmt.Errorf("check house ownership: %w", err)
	}
	if !owned {
		return nil, model.ErrRoomNotFound
	}

	house, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID)
	if err != nil {
		return nil, fmt.Errorf("get house for invoice: %w", err)
	}

	oldElecIndex := 0
	oldWaterIndex := 0

	// Get the previous invoice to get old indices
	prevInvoice, err := s.invoiceRepo.GetPreviousInvoice(ctx, input.RoomID, input.Period)
	if err != nil {
		if !errors.Is(err, model.ErrInvoiceNotFound) {
			return nil, fmt.Errorf("get previous invoice: %w", err)
		}
	} else {
		oldElecIndex = prevInvoice.NewElectricityIndex
		oldWaterIndex = prevInvoice.NewWaterIndex
	}

	if input.OldElectricityIndex != nil {
		oldElecIndex = *input.OldElectricityIndex
	}
	if input.OldWaterIndex != nil {
		oldWaterIndex = *input.OldWaterIndex
	}

	tenantCountInt := 0
	if input.TenantCount != nil {
		tenantCountInt = *input.TenantCount
	} else {
		tenantCount, err := s.tenantRepo.GetCurrentNumTenantInRoom(ctx, input.RoomID)
		if err != nil {
			return nil, fmt.Errorf("get tenant count for extra fee: %w", err)
		}
		tenantCountInt = int(tenantCount)
	}

	invoice, err := s.calculateInvoice(room, house, input, oldElecIndex, oldWaterIndex, tenantCountInt)
	if err != nil {
		return nil, err
	}

	// Check if invoice for this period already exists
	existingInvoice, err := s.invoiceRepo.GetInvoiceByRoomAndPeriod(ctx, input.RoomID, input.Period)
	if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
		return nil, fmt.Errorf("check existing invoice: %w", err)
	}

	if existingInvoice != nil {
		if existingInvoice.Status == "PAID" {
			return nil, model.ErrPaidInvoiceImmutable
		}
		invoice.ID = existingInvoice.ID
		invoice.CreatedAt = existingInvoice.CreatedAt
		err = s.invoiceRepo.UpdateInvoice(ctx, managerID, invoice)
		if err != nil {
			return nil, err
		}
	} else {
		err = s.invoiceRepo.CreateInvoice(ctx, invoice)
		if err != nil {
			return nil, err
		}
	}

	if s.eventBus != nil {
		s.eventBus.Publish(revenue.EventInvoiceChanged, revenue.RevenueSummaryPayload{
			HouseID: room.HouseID,
			Period:  input.Period,
		})
	}

	return s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoice.ID)
}

func normalizeOtherFees(items []model.InvoiceFeeItem, legacyAmount float64) ([]model.InvoiceFeeItem, float64) {
	if items == nil {
		return []model.InvoiceFeeItem{}, legacyAmount
	}
	normalized := make([]model.InvoiceFeeItem, 0, len(items))
	total := 0.0
	for _, item := range items {
		if item.Amount <= 0 {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = "Chi phí phát sinh"
		}
		normalized = append(normalized, model.InvoiceFeeItem{Name: name, Amount: item.Amount})
		total += item.Amount
	}
	return normalized, total
}

func (s *InvoiceServiceImpl) GetInvoice(ctx context.Context, managerID, invoiceID string) (*model.InvoiceWithRoom, error) {
	return s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
}

func (s *InvoiceServiceImpl) ListInvoices(ctx context.Context, managerID string, filter model.InvoiceListFilter) ([]model.InvoiceWithRoom, int, error) {
	return s.invoiceRepo.ListInvoices(ctx, managerID, filter)
}

func (s *InvoiceServiceImpl) SumInvoicesByStatus(ctx context.Context, managerID string, filter model.InvoiceBreakdownFilter) ([]model.InvoiceStatusTotals, error) {
	return s.invoiceRepo.SumInvoicesByStatus(ctx, managerID, filter)
}

func (s *InvoiceServiceImpl) PayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == "PAID" {
		return nil, model.ErrInvoiceAlreadyPaid
	}

	res, err := s.invoiceRepo.UpdateInvoiceStatus(ctx, managerID, invoiceID, "PAID")
	if err == nil && s.eventBus != nil {
		s.eventBus.Publish(revenue.EventInvoiceChanged, revenue.RevenueSummaryPayload{
			HouseID: invoice.HouseID,
			Period:  invoice.Period,
		})
	}
	return res, err
}

func (s *InvoiceServiceImpl) UnpayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == "UNPAID" {
		return nil, model.ErrInvoiceAlreadyUnpaid
	}

	res, err := s.invoiceRepo.UpdateInvoiceStatus(ctx, managerID, invoiceID, "UNPAID")
	if err == nil && s.eventBus != nil {
		s.eventBus.Publish(revenue.EventInvoiceChanged, revenue.RevenueSummaryPayload{
			HouseID: invoice.HouseID,
			Period:  invoice.Period,
		})
	}
	return res, err
}

func (s *InvoiceServiceImpl) DeleteInvoice(ctx context.Context, managerID, invoiceID string) error {
	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status == "PAID" {
		return model.ErrPaidInvoiceDelete
	}

	err = s.invoiceRepo.DeleteInvoice(ctx, managerID, invoiceID)
	if err == nil && s.eventBus != nil {
		s.eventBus.Publish(revenue.EventInvoiceChanged, revenue.RevenueSummaryPayload{
			HouseID: invoice.HouseID,
			Period:  invoice.Period,
		})
	}
	return err
}

// ResolveTransactionImagePath verifies invoice ownership before returning a local upload path.
func (s *InvoiceServiceImpl) ResolveTransactionImagePath(ctx context.Context, managerID, requestPath string) (string, error) {
	fileName, ok := shared.UploadFileName(requestPath)
	if !ok {
		return "", shared.ErrInvalidInput
	}

	storedPath := "/uploads/transactions/" + fileName
	if _, err := s.invoiceRepo.GetInvoiceByTransactionImagePath(ctx, managerID, storedPath); err != nil {
		return "", err
	}

	filePath, ok := shared.UploadFilePath("uploads/transactions", fileName)
	if !ok {
		return "", shared.ErrInvalidInput
	}
	return filePath, nil
}

func (s *InvoiceServiceImpl) calculateUtilityFee(billingType, billingUnit string, defaultPrice float64, newIndex, oldIndex int, tenantCount int) (float64, error) {
	if billingType == "FIXED" {
		if billingUnit == "PERSON" {
			return defaultPrice * float64(tenantCount), nil
		}
		// ROOM: flat price per room
		return defaultPrice, nil
	}

	// USAGE: calculate by meter reading difference
	return float64(newIndex-oldIndex) * defaultPrice, nil
}

type RecalculationError struct {
	Succeeded int
	Failed    int
	Cause     error
}

func (e *RecalculationError) Error() string {
	return fmt.Sprintf("invoice recalculation: %d succeeded, %d failed: %v", e.Succeeded, e.Failed, e.Cause)
}

func (e *RecalculationError) Unwrap() error { return e.Cause }

func (s *InvoiceServiceImpl) RecalculateUnpaidInvoicesByRoom(ctx context.Context, managerID, roomID string) error {
	room, err := s.roomRepo.GetRoomByIDForManager(ctx, managerID, roomID)
	if err != nil {
		return err
	}
	house, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID)
	if err != nil {
		return err
	}
	invoices, err := s.invoiceRepo.GetUnpaidInvoicesByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	return s.recalculateInvoices(ctx, managerID, house, map[string]*model.Room{room.ID: room}, invoices)
}

func (s *InvoiceServiceImpl) RecalculateUnpaidInvoicesByHouse(ctx context.Context, managerID, houseID string) error {
	house, err := s.houseRepo.GetByID(ctx, houseID, managerID)
	if err != nil {
		return err
	}
	rooms, err := s.roomRepo.ListAllRoomsByHouseID(ctx, houseID)
	if err != nil {
		return err
	}
	invoices, err := s.invoiceRepo.GetUnpaidInvoicesByHouseID(ctx, managerID, houseID)
	if err != nil {
		return err
	}
	byID := make(map[string]*model.Room, len(rooms))
	for i := range rooms {
		byID[rooms[i].ID] = &rooms[i]
	}
	return s.recalculateInvoices(ctx, managerID, house, byID, invoices)
}

func (s *InvoiceServiceImpl) recalculateInvoices(ctx context.Context, managerID string, house *model.House, rooms map[string]*model.Room, invoices []model.Invoice) error {
	result := &RecalculationError{}
	for _, inv := range invoices {
		if err := ctx.Err(); err != nil {
			return errors.Join(result.Cause, err)
		}
		room := rooms[inv.RoomID]
		var err error
		if room == nil {
			err = model.ErrRoomNotFound
		} else {
			excludeRoomFee := inv.RoomFee == 0
			input := CreateInvoiceInput{
				RoomID: inv.RoomID, Period: inv.Period,
				NewElectricityIndex: inv.NewElectricityIndex, NewWaterIndex: inv.NewWaterIndex,
				OtherFee: inv.OtherFee, OtherFees: inv.OtherFees, Discount: inv.Discount,
				VehicleCount: inv.VehicleCount, ExcludeRoomFee: &excludeRoomFee,
			}
			var updated *model.Invoice
			updated, err = s.calculateInvoice(room, house, input, inv.OldElectricityIndex, inv.OldWaterIndex, inv.TenantCount)
			if err == nil {
				updated.ID = inv.ID
				updated.CreatedAt = inv.CreatedAt
				updated.Status = inv.Status
				updated.PaymentMethod = inv.PaymentMethod
				updated.TransactionImagePath = inv.TransactionImagePath
				err = s.invoiceRepo.UpdateInvoice(ctx, managerID, updated)
			}
		}
		if err != nil {
			result.Failed++
			result.Cause = errors.Join(result.Cause, fmt.Errorf("invoice %s: %w", inv.ID, err))
		} else {
			result.Succeeded++
		}
	}
	if result.Failed > 0 {
		return result
	}
	return nil
}

func (s *InvoiceServiceImpl) calculateInvoice(room *model.Room, house *model.House, input CreateInvoiceInput, oldElecIndex, oldWaterIndex, tenantCountInt int) (*model.Invoice, error) {
	elecPrice := house.DefaultElectricityPrice
	if room.ElectricityPrice != nil {
		elecPrice = *room.ElectricityPrice
	}

	waterPrice := house.DefaultWaterPrice
	if room.WaterPrice != nil {
		waterPrice = *room.WaterPrice
	}

	wifiPrice := house.DefaultWifiPrice
	if room.WifiPrice != nil {
		wifiPrice = *room.WifiPrice
	}

	parkingPrice := house.DefaultParkingPrice
	if room.ParkingPrice != nil {
		parkingPrice = *room.ParkingPrice
	}

	servicePrice := house.DefaultServicePrice
	if room.ServicePrice != nil {
		servicePrice = *room.ServicePrice
	}

	// Calculate electricity fee based on billing type
	elecFee, err := s.calculateUtilityFee(house.ElectricityBillingType, house.ElectricityBillingUnit, elecPrice, input.NewElectricityIndex, oldElecIndex, tenantCountInt)
	if err != nil {
		return nil, err
	}

	// Calculate water fee based on billing type
	waterFee, err := s.calculateUtilityFee(house.WaterBillingType, house.WaterBillingUnit, waterPrice, input.NewWaterIndex, oldWaterIndex, tenantCountInt)
	if err != nil {
		return nil, err
	}

	if house.ElectricityBillingType == "FIXED" {
		input.NewElectricityIndex = oldElecIndex
	} else if input.NewElectricityIndex < oldElecIndex {
		return nil, fmt.Errorf("%w (số cũ: %d)", model.ErrInvalidElectricityIndex, oldElecIndex)
	}

	if house.WaterBillingType == "FIXED" {
		input.NewWaterIndex = oldWaterIndex
	} else if input.NewWaterIndex < oldWaterIndex {
		return nil, fmt.Errorf("%w (số cũ: %d)", model.ErrInvalidWaterIndex, oldWaterIndex)
	}

	roomFee := float64(room.Price)
	excludeRoomFee := room.Status == "AVAILABLE"
	if input.ExcludeRoomFee != nil {
		excludeRoomFee = *input.ExcludeRoomFee
	}
	if excludeRoomFee {
		roomFee = 0
	}

	// Resolve surcharge thresholds: room override takes priority over house default
	personThreshold := house.ExtraPersonThreshold
	if room.ExtraPersonThreshold != nil {
		personThreshold = *room.ExtraPersonThreshold
	}
	personFeeUnit := house.ExtraPersonFee
	if room.ExtraPersonFee != nil {
		personFeeUnit = *room.ExtraPersonFee
	}
	vehicleThreshold := house.ExtraVehicleThreshold
	if room.ExtraVehicleThreshold != nil {
		vehicleThreshold = *room.ExtraVehicleThreshold
	}
	vehicleFeeUnit := house.ExtraVehicleFee
	if room.ExtraVehicleFee != nil {
		vehicleFeeUnit = *room.ExtraVehicleFee
	}

	extraPersonFee := 0.0
	if personThreshold > 0 && tenantCountInt > personThreshold {
		extraPersonFee = float64(tenantCountInt-personThreshold) * personFeeUnit
	}

	extraVehicleFee := 0.0
	if vehicleThreshold > 0 && input.VehicleCount > vehicleThreshold {
		extraVehicleFee = float64(input.VehicleCount-vehicleThreshold) * vehicleFeeUnit
	}

	otherFees, otherFee := normalizeOtherFees(input.OtherFees, input.OtherFee)

	totalAmount := roomFee + elecFee + waterFee + wifiPrice + parkingPrice + servicePrice + extraPersonFee + extraVehicleFee + otherFee - input.Discount

	invoice := &model.Invoice{
		ID:                  uuid.Must(uuid.NewV7()).String(),
		RoomID:              room.ID,
		Period:              input.Period,
		RoomFee:             roomFee,
		OldElectricityIndex: oldElecIndex,
		NewElectricityIndex: input.NewElectricityIndex,
		ElectricityFee:      elecFee,
		OldWaterIndex:       oldWaterIndex,
		NewWaterIndex:       input.NewWaterIndex,
		WaterFee:            waterFee,
		WifiFee:             wifiPrice,
		ParkingFee:          parkingPrice,
		ServiceFee:          servicePrice,
		OtherFee:            otherFee,
		OtherFees:           otherFees,
		Discount:            input.Discount,
		TenantCount:         tenantCountInt,
		VehicleCount:        input.VehicleCount,
		ExtraPersonFee:      extraPersonFee,
		ExtraVehicleFee:     extraVehicleFee,
		TotalAmount:         totalAmount,
		Status:              "UNPAID",
	}

	return invoice, nil
}
