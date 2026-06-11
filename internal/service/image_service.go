package service

import (
	"bytes"
	"context"
	"fmt"
	"image/png"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/mihb123/quanly-phongtro/internal/assets"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type ImageService interface {
	GenerateInvoiceImage(ctx context.Context, invoice *model.InvoiceWithRoom) ([]byte, error)
}

type imageServiceImpl struct{}

func NewImageService() ImageService {
	return &imageServiceImpl{}
}

func formatCurrencyToVND(amount float64) string {
	p := message.NewPrinter(language.Vietnamese)
	return p.Sprintf("%.0f ₫", amount)
}

func loadFont(dc *gg.Context, fontName string, points float64) error {
	var fontBytes []byte
	if fontName == "Roboto-Bold.ttf" {
		fontBytes = assets.RobotoBold
	} else if fontName == "Roboto-Regular.ttf" {
		fontBytes = assets.RobotoRegular
	} else {
		return fmt.Errorf("unknown font: %s", fontName)
	}

	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("failed to parse font: %w", err)
	}

	face := truetype.NewFace(f, &truetype.Options{
		Size: points,
	})
	dc.SetFontFace(face)
	return nil
}

type invoiceLine struct {
	label      string
	desc       string
	value      float64
	isDiscount bool
}

func (s *imageServiceImpl) GenerateInvoiceImage(ctx context.Context, invoice *model.InvoiceWithRoom) ([]byte, error) {
	showUtilityTable := invoice.ElectricityFee > 0 || invoice.WaterFee > 0

	type utilityLine struct {
		name     string
		oldIdx   string
		newIdx   string
		usageStr string
		priceStr string
		totalFee float64
	}
	var utilLines []utilityLine

	getUtilDisplay := func(fee float64, usage int, unit string) (string, string) {
		if usage > 0 {
			return fmt.Sprintf("%d %s", usage, unit), formatCurrencyToVND(fee / float64(usage))
		}
		if fee > 0 {
			if invoice.TenantCount > 0 && int(fee)%invoice.TenantCount == 0 && (fee/float64(invoice.TenantCount)) >= 1000 {
				return fmt.Sprintf("%d người", invoice.TenantCount), formatCurrencyToVND(fee / float64(invoice.TenantCount))
			}
			return "Khoán", formatCurrencyToVND(fee)
		}
		return "-", "-"
	}

	if invoice.ElectricityFee > 0 {
		usage := invoice.NewElectricityIndex - invoice.OldElectricityIndex
		oldIdx := "-"
		newIdx := "-"
		if usage > 0 {
			oldIdx = fmt.Sprintf("%d", invoice.OldElectricityIndex)
			newIdx = fmt.Sprintf("%d", invoice.NewElectricityIndex)
		}
		usageStr, priceStr := getUtilDisplay(invoice.ElectricityFee, usage, "kWh")
		utilLines = append(utilLines, utilityLine{"Điện", oldIdx, newIdx, usageStr, priceStr, invoice.ElectricityFee})
	}
	if invoice.WaterFee > 0 {
		usage := invoice.NewWaterIndex - invoice.OldWaterIndex
		oldIdx := "-"
		newIdx := "-"
		if usage > 0 {
			oldIdx = fmt.Sprintf("%d", invoice.OldWaterIndex)
			newIdx = fmt.Sprintf("%d", invoice.NewWaterIndex)
		}
		usageStr, priceStr := getUtilDisplay(invoice.WaterFee, usage, "m³")
		utilLines = append(utilLines, utilityLine{"Nước", oldIdx, newIdx, usageStr, priceStr, invoice.WaterFee})
	}

	var lines []invoiceLine
	if invoice.RoomFee > 0 {
		lines = append(lines, invoiceLine{"Tiền phòng", "", invoice.RoomFee, false})
	}
	if invoice.ElectricityFee > 0 {
		usage := invoice.NewElectricityIndex - invoice.OldElectricityIndex
		var desc string
		if usage > 0 {
			desc = fmt.Sprintf("Tiêu thụ: %d kWh", usage)
		} else {
			desc = ""
		}
		lines = append(lines, invoiceLine{"Tiền điện", desc, invoice.ElectricityFee, false})
	}
	if invoice.WaterFee > 0 {
		usage := invoice.NewWaterIndex - invoice.OldWaterIndex
		var desc string
		if usage > 0 {
			desc = fmt.Sprintf("Tiêu thụ: %d m³", usage)
		} else {
			desc = ""
		}
		lines = append(lines, invoiceLine{"Tiền nước", desc, invoice.WaterFee, false})
	}
	if invoice.WifiFee > 0 {
		lines = append(lines, invoiceLine{"Tiền mạng Wifi", "", invoice.WifiFee, false})
	}
	if invoice.ParkingFee > 0 {
		lines = append(lines, invoiceLine{"Tiền gửi xe", fmt.Sprintf("Gửi %d xe máy", invoice.VehicleCount), invoice.ParkingFee, false})
	}
	if invoice.ServiceFee > 0 {
		lines = append(lines, invoiceLine{"Phí dịch vụ chung", "Vệ sinh, rác, giặt sấy", invoice.ServiceFee, false})
	}
	if invoice.ExtraPersonFee > 0 {
		lines = append(lines, invoiceLine{
			"Phụ thu người thêm",
			fmt.Sprintf("%d người vượt đ.mức (%s/ng)", invoice.TenantCount-invoice.ExtraPersonThreshold, formatCurrencyToVND(invoice.ExtraPersonFeeUnit)),
			invoice.ExtraPersonFee,
			false,
		})
	}
	if invoice.ExtraVehicleFee > 0 {
		lines = append(lines, invoiceLine{
			"Phụ thu xe thêm",
			fmt.Sprintf("%d xe vượt đ.mức (%s/xe)", invoice.VehicleCount-invoice.ExtraVehicleThreshold, formatCurrencyToVND(invoice.ExtraVehicleFeeUnit)),
			invoice.ExtraVehicleFee,
			false,
		})
	}
	if invoice.OtherFee > 0 {
		lines = append(lines, invoiceLine{"Chi phí phát sinh", "Chi phí khác phát sinh", invoice.OtherFee, false})
	}
	if invoice.Discount > 0 {
		lines = append(lines, invoiceLine{"Giảm trừ", "Giảm giá", invoice.Discount, true})
	}

	yTable2 := 215.0
	var yRow float64
	if showUtilityTable {
		yRow = 278.0 + 32.0*float64(len(utilLines))
		yTable2 = yRow + 20.0
	}
	yStart := yTable2 + 58.0
	yEndTable2 := yStart + float64(len(lines))*32.0
	yTotal := yEndTable2 + 15.0
	neededHeight := int(yTotal + 75.0 + 40.0)

	dc := gg.NewContext(600, neededHeight)

	dc.SetHexColor("#f8fafc")
	dc.Clear()

	dc.SetHexColor("#cbd5e1")
	dc.DrawRoundedRectangle(24, 24, 552, float64(neededHeight-48), 16)
	dc.Fill()

	dc.SetHexColor("#ffffff")
	dc.DrawRoundedRectangle(25, 25, 550, float64(neededHeight-50), 16)
	dc.Fill()

	dc.SetHexColor("#1e1b4b")
	if err := loadFont(dc, "Roboto-Bold.ttf", 26); err == nil {
		dc.DrawStringAnchored("HÓA ĐƠN TIỀN NHÀ", 300, 70, 0.5, 0.5)
	}

	dc.SetHexColor("#64748b")
	if err := loadFont(dc, "Roboto-Regular.ttf", 15); err == nil {
		dc.DrawStringAnchored(fmt.Sprintf("Kỳ hóa đơn: %s", invoice.Period), 300, 100, 0.5, 0.5)
	}

	dc.SetHexColor("#f1f5f9")
	dc.DrawRoundedRectangle(50, 130, 500, 50, 8)
	dc.Fill()

	dc.SetHexColor("#0f172a")
	if err := loadFont(dc, "Roboto-Bold.ttf", 17); err == nil {
		dc.DrawStringAnchored(fmt.Sprintf("Phòng: %s", invoice.RoomName), 300, 155, 0.5, 0.5)
	}

	if showUtilityTable {
		dc.SetHexColor("#4f46e5")
		_ = loadFont(dc, "Roboto-Bold.ttf", 13)
		dc.DrawString("1. BẢNG KÊ CHỈ SỐ ĐIỆN / NƯỚC", 60, 220)

		dc.SetHexColor("#64748b")
		_ = loadFont(dc, "Roboto-Bold.ttf", 10)
		dc.DrawString("DỊCH VỤ", 60, 242)
		dc.DrawStringAnchored("CHỈ SỐ CŨ", 180, 242, 0.5, 0)
		dc.DrawStringAnchored("CHỈ SỐ MỚI", 260, 242, 0.5, 0)
		dc.DrawStringAnchored("TIÊU THỤ", 340, 242, 0.5, 0)
		dc.DrawStringAnchored("ĐƠN GIÁ", 440, 242, 1.0, 0)
		dc.DrawStringAnchored("THÀNH TIỀN", 540, 242, 1.0, 0)

		dc.SetHexColor("#e2e8f0")
		dc.SetLineWidth(1)
		dc.DrawLine(50, 256, 550, 256)
		dc.Stroke()

		yRow = 278.0
		for _, ul := range utilLines {
			dc.SetHexColor("#0f172a")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawString(ul.name, 60, yRow)

			dc.SetHexColor("#334155")
			_ = loadFont(dc, "Roboto-Regular.ttf", 13)
			dc.DrawStringAnchored(ul.oldIdx, 180, yRow, 0.5, 0)
			dc.DrawStringAnchored(ul.newIdx, 260, yRow, 0.5, 0)
			dc.DrawStringAnchored(ul.usageStr, 340, yRow, 0.5, 0)
			dc.DrawStringAnchored(ul.priceStr, 440, yRow, 1.0, 0)

			dc.SetHexColor("#0f172a")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawStringAnchored(formatCurrencyToVND(ul.totalFee), 540, yRow, 1.0, 0)

			dc.SetHexColor("#f1f5f9")
			dc.DrawLine(50, yRow+12, 550, yRow+12)
			dc.Stroke()

			yRow += 32.0
		}

		dc.SetHexColor("#cbd5e1")
		dc.DrawLine(50, yRow-10, 550, yRow-10)
		dc.Stroke()
	}

	dc.SetHexColor("#4f46e5")
	_ = loadFont(dc, "Roboto-Bold.ttf", 13)
	dc.DrawString("2. CHI TIẾT CÁC CHI PHÍ DỊCH VỤ", 60, yTable2)

	dc.SetHexColor("#64748b")
	_ = loadFont(dc, "Roboto-Bold.ttf", 10)
	dc.DrawString("KHOẢN MỤC DỊCH VỤ", 60, yTable2+22)
	dc.DrawStringAnchored("THÀNH TIỀN", 540, yTable2+22, 1.0, 0)

	dc.SetHexColor("#cbd5e1")
	dc.SetLineWidth(1)
	dc.DrawLine(50, yTable2+36, 550, yTable2+36)
	dc.Stroke()

	lineGap := 32.0
	for _, line := range lines {
		dc.SetHexColor("#0f172a")
		_ = loadFont(dc, "Roboto-Bold.ttf", 13)
		dc.DrawString(line.label, 60, yStart)

		if line.isDiscount {
			dc.SetHexColor("#dc2626")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawStringAnchored(fmt.Sprintf("-%s", formatCurrencyToVND(line.value)), 540, yStart, 1.0, 0)
		} else {
			dc.SetHexColor("#0f172a")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawStringAnchored(formatCurrencyToVND(line.value), 540, yStart, 1.0, 0)
		}

		dc.SetHexColor("#f1f5f9")
		dc.SetLineWidth(1)
		dc.DrawLine(50, yStart+16, 550, yStart+16)
		dc.Stroke()

		yStart += lineGap
	}

	dc.SetHexColor("#e0e7ff")
	dc.DrawRoundedRectangle(50, yTotal, 500, 75, 10)
	dc.Fill()

	dc.SetHexColor("#3730a3")
	_ = loadFont(dc, "Roboto-Bold.ttf", 16)
	dc.DrawString("TỔNG TIỀN CẦN THANH TOÁN", 60, yTotal+46)

	dc.SetHexColor("#4f46e5")
	_ = loadFont(dc, "Roboto-Bold.ttf", 24)
	dc.DrawStringAnchored(formatCurrencyToVND(invoice.TotalAmount), 540, yTotal+46, 1.0, 0.0)

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
