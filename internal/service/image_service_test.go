package service_test

import (
	"context"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

func TestImageService_GenerateInvoiceImage(t *testing.T) {
	svc := service.NewImageService()

	// Smoke test with basic data to ensure it doesn't panic and returns valid PNG bytes
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:                  "inv123",
			RoomID:              "room123",
			Period:              "2023-10",
			RoomFee:             2500000,
			OldElectricityIndex: 100,
			NewElectricityIndex: 250,
			ElectricityFee:      525000, // (250-100) * 3500
			OldWaterIndex:       10,
			NewWaterIndex:       15,
			WaterFee:            100000, // (15-10) * 20000
			WifiFee:             100000,
			ParkingFee:          200000,
			ServiceFee:          50000,
			ExtraPersonFee:      0,
			ExtraVehicleFee:     0,
			OtherFee:            0,
			Discount:            50000,
			TotalAmount:         3425000,
			Status:              "UNPAID",
			VehicleCount:        2,
			TenantCount:         2,
		},
		RoomName: "P101",
		HouseID:  "house123",
	}

	ctx := context.Background()
	imgBytes, err := svc.GenerateInvoiceImage(ctx, invoice)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(imgBytes) == 0 {
		t.Fatalf("expected image bytes, got empty slice")
	}

	// Basic check that it contains a PNG header
	// PNG signature: 137 80 78 71 13 10 26 10
	if len(imgBytes) > 8 {
		signature := []byte{137, 80, 78, 71, 13, 10, 26, 10}
		for i := 0; i < 8; i++ {
			if imgBytes[i] != signature[i] {
				t.Fatalf("expected PNG signature, but mismatch at index %d", i)
			}
		}
	} else {
		t.Fatalf("returned image bytes are too short to be a valid PNG")
	}
}
