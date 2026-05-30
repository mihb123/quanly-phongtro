package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type CreateInvoiceInput struct {
	RoomID              string
	Period              string
	NewElectricityIndex int
	NewWaterIndex       int
	OtherFee            float64
	Discount            float64
	VehicleCount        int
}

type InvoiceService interface {
	CreateInvoice(ctx context.Context, managerID string, input CreateInvoiceInput) (*model.InvoiceWithRoom, error)
	GetInvoice(ctx context.Context, managerID, invoiceID string) (*model.InvoiceWithRoom, error)
	ListInvoices(ctx context.Context, managerID string, filter model.InvoiceListFilter) ([]model.InvoiceWithRoom, error)
	PayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error)
	UnpayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error)
	RecalculateUnpaidInvoicesByRoom(ctx context.Context, managerID, roomID string) error
	RecalculateUnpaidInvoicesByHouse(ctx context.Context, managerID, houseID string) error
}

type InvoiceServiceImpl struct {
	invoiceRepo model.InvoiceRepository
	roomRepo    model.RoomRepository
	houseRepo   model.HouseRepository
	tenantRepo  model.TenantRepository
}

func NewInvoiceService(invoiceRepo model.InvoiceRepository, roomRepo model.RoomRepository, houseRepo model.HouseRepository, tenantRepo model.TenantRepository) InvoiceService {
	return &InvoiceServiceImpl{
		invoiceRepo: invoiceRepo,
		roomRepo:    roomRepo,
		houseRepo:   houseRepo,
		tenantRepo:  tenantRepo,
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

	tenantCount, err := s.tenantRepo.GetCurrentNumTenantInRoom(ctx, input.RoomID)
	if err != nil {
		return nil, fmt.Errorf("get tenant count for extra fee: %w", err)
	}
	tenantCountInt := int(tenantCount)

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

	// For FIXED billing, indices are not meaningful — keep them equal so usage = 0
	if house.ElectricityBillingType == "FIXED" {
		input.NewElectricityIndex = oldElecIndex
	} else if input.NewElectricityIndex < oldElecIndex {
		return nil, model.ErrInvalidElectricityIndex
	}

	if house.WaterBillingType == "FIXED" {
		input.NewWaterIndex = oldWaterIndex
	} else if input.NewWaterIndex < oldWaterIndex {
		return nil, model.ErrInvalidWaterIndex
	}

	roomFee := float64(room.Price)

	// tenantCount already fetched above

	extraPersonFee := 0.0
	if house.ExtraPersonThreshold > 0 && tenantCountInt > house.ExtraPersonThreshold {
		extraPersonFee = float64(tenantCountInt-house.ExtraPersonThreshold) * house.ExtraPersonFee
	}

	extraVehicleFee := 0.0
	if house.ExtraVehicleThreshold > 0 && input.VehicleCount > house.ExtraVehicleThreshold {
		extraVehicleFee = float64(input.VehicleCount-house.ExtraVehicleThreshold) * house.ExtraVehicleFee
	}

	totalAmount := roomFee + elecFee + waterFee + wifiPrice + parkingPrice + servicePrice + extraPersonFee + extraVehicleFee + input.OtherFee - input.Discount

	invoice := &model.Invoice{
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
		OtherFee:            input.OtherFee,
		Discount:            input.Discount,
		TenantCount:         tenantCountInt,
		VehicleCount:        input.VehicleCount,
		ExtraPersonFee:      extraPersonFee,
		ExtraVehicleFee:     extraVehicleFee,
		TotalAmount:         totalAmount,
		Status:              "UNPAID",
	}

	// Check if invoice for this period already exists
	existingInvoice, err := s.invoiceRepo.GetInvoiceByRoomAndPeriod(ctx, input.RoomID, input.Period)
	if err != nil && !errors.Is(err, model.ErrInvoiceNotFound) {
		return nil, fmt.Errorf("check existing invoice: %w", err)
	}

	if existingInvoice != nil {
		if existingInvoice.Status == "PAID" {
			return nil, errors.New("cannot edit a paid invoice")
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

	return s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoice.ID)
}

func (s *InvoiceServiceImpl) GetInvoice(ctx context.Context, managerID, invoiceID string) (*model.InvoiceWithRoom, error) {
	return s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
}

func (s *InvoiceServiceImpl) ListInvoices(ctx context.Context, managerID string, filter model.InvoiceListFilter) ([]model.InvoiceWithRoom, error) {
	return s.invoiceRepo.ListInvoices(ctx, managerID, filter)
}

func (s *InvoiceServiceImpl) PayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == "PAID" {
		return nil, errors.New("invoice is already paid")
	}

	return s.invoiceRepo.UpdateInvoiceStatus(ctx, managerID, invoiceID, "PAID")
}

func (s *InvoiceServiceImpl) UnpayInvoice(ctx context.Context, managerID, invoiceID string) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == "UNPAID" {
		return nil, errors.New("invoice is already unpaid")
	}

	return s.invoiceRepo.UpdateInvoiceStatus(ctx, managerID, invoiceID, "UNPAID")
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

func (s *InvoiceServiceImpl) RecalculateUnpaidInvoicesByRoom(ctx context.Context, managerID, roomID string) error {
	invoices, err := s.invoiceRepo.GetUnpaidInvoicesByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	for _, inv := range invoices {
		input := CreateInvoiceInput{
			RoomID:              inv.RoomID,
			Period:              inv.Period,
			NewElectricityIndex: inv.NewElectricityIndex,
			NewWaterIndex:       inv.NewWaterIndex,
			OtherFee:            inv.OtherFee,
			Discount:            inv.Discount,
			VehicleCount:        inv.VehicleCount,
		}
		// CreateInvoice acts as an upsert for the same room and period
		_, err := s.CreateInvoice(ctx, managerID, input)
		if err != nil {
			fmt.Printf("recalculate invoice roomID=%s period=%s: %v\n", inv.RoomID, inv.Period, err)
		}
	}
	return nil
}

func (s *InvoiceServiceImpl) RecalculateUnpaidInvoicesByHouse(ctx context.Context, managerID, houseID string) error {
	rooms, err := s.roomRepo.ListAllRoomsByHouseID(ctx, houseID)
	if err != nil {
		return err
	}
	for _, room := range rooms {
		err := s.RecalculateUnpaidInvoicesByRoom(ctx, managerID, room.ID)
		if err != nil {
			fmt.Printf("recalculate unpaid invoices for roomID=%s: %v\n", room.ID, err)
		}
	}
	return nil
}
