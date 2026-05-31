package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"fmt"
	"image/png"

	"github.com/fogleman/gg"
	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type InvoiceHandler struct {
	invoiceService service.InvoiceService
}

func NewInvoiceHandler(invoiceService service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

type createInvoiceRequest struct {
	RoomID              string  `json:"room_id" validate:"required"`
	Period              string  `json:"period" validate:"required"`
	NewElectricityIndex int     `json:"new_electricity_index" validate:"gte=0"`
	NewWaterIndex       int     `json:"new_water_index" validate:"gte=0"`
	OtherFee            float64 `json:"other_fee" validate:"gte=0"`
	Discount            float64 `json:"discount" validate:"gte=0"`
	VehicleCount        int     `json:"vehicle_count" validate:"gte=0"`
}

func handleInvoiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrInvoiceNotFound):
		logger.Warn(r, http.StatusNotFound, "invoice not found", err)
		writeError(w, http.StatusNotFound, "invoice not found")
	case errors.Is(err, model.ErrRoomNotFound):
		logger.Warn(r, http.StatusNotFound, "room not found", err)
		writeError(w, http.StatusNotFound, "room not found")
	case errors.Is(err, model.ErrDuplicateInvoice):
		logger.Warn(r, http.StatusConflict, "duplicate invoice", err)
		writeError(w, http.StatusConflict, "duplicate invoice for this room and period")
	case errors.Is(err, model.ErrInvalidElectricityIndex), errors.Is(err, model.ErrInvalidWaterIndex):
		logger.Warn(r, http.StatusBadRequest, "invalid indices", err)
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		logger.Error(r, http.StatusInternalServerError, "internal server error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	var req createInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request format")
		return
	}
	defer r.Body.Close()

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusUnprocessableEntity, "validation failed", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	input := service.CreateInvoiceInput{
		RoomID:              req.RoomID,
		Period:              req.Period,
		NewElectricityIndex: req.NewElectricityIndex,
		NewWaterIndex:       req.NewWaterIndex,
		OtherFee:            req.OtherFee,
		Discount:            req.Discount,
		VehicleCount:        req.VehicleCount,
	}

	invoice, err := h.invoiceService.CreateInvoice(r.Context(), managerID, input)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	filter := model.InvoiceListFilter{
		HouseID: r.URL.Query().Get("house_id"),
		RoomID:  r.URL.Query().Get("room_id"),
		Period:  r.URL.Query().Get("period"),
		Status:  r.URL.Query().Get("status"),
		Page:    1,
		Limit:   20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	invoices, err := h.invoiceService.ListInvoices(r.Context(), managerID, filter)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoices); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.PayInvoice(r.Context(), managerID, id)
	if err != nil {
		if err.Error() == "invoice is already paid" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) UnpayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.UnpayInvoice(r.Context(), managerID, id)
	if err != nil {
		if err.Error() == "invoice is already unpaid" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func formatCurrencyToVND(amount float64) string {
	p := message.NewPrinter(language.Vietnamese)
	return p.Sprintf("%.0f ₫", amount)
}

func loadFont(dc *gg.Context, fontName string, points float64) error {
	paths := []string{
		"assets/" + fontName,
		"../assets/" + fontName,
		"../../assets/" + fontName,
		"/media/minhchu1336/Data/quanly-phongtro/assets/" + fontName,
	}
	var lastErr error
	for _, p := range paths {
		if err := dc.LoadFontFace(p, points); err == nil {
			return nil
		}
		lastErr = fmt.Errorf("failed to load font from %s: %w", p, lastErr)
	}
	return lastErr
}

type invoiceLine struct {
	label      string
	desc       string
	value      float64
	isDiscount bool
}

func (h *InvoiceHandler) DownloadInvoiceImage(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	// Dynamic variables for tables
	showUtilityTable := false
	type utilityLine struct {
		name      string
		oldIdx    int
		newIdx    int
		usage     int
		unit      string
		unitPrice float64
	}
	var utilLines []utilityLine

	if invoice.ElectricityFee > 0 {
		usage := invoice.NewElectricityIndex - invoice.OldElectricityIndex
		if usage > 0 {
			showUtilityTable = true
			unitPrice := invoice.ElectricityFee / float64(usage)
			utilLines = append(utilLines, utilityLine{"Điện", invoice.OldElectricityIndex, invoice.NewElectricityIndex, usage, "kWh", unitPrice})
		}
	}
	if invoice.WaterFee > 0 {
		usage := invoice.NewWaterIndex - invoice.OldWaterIndex
		if usage > 0 {
			showUtilityTable = true
			unitPrice := invoice.WaterFee / float64(usage)
			utilLines = append(utilLines, utilityLine{"Nước", invoice.OldWaterIndex, invoice.NewWaterIndex, usage, "m³", unitPrice})
		}
	}

	// structured invoice line items
	var lines []invoiceLine

	if invoice.RoomFee > 0 {
		lines = append(lines, invoiceLine{"Tiền thuê phòng", "Giá thuê phòng cố định", invoice.RoomFee, false})
	}
	if invoice.ElectricityFee > 0 {
		usage := invoice.NewElectricityIndex - invoice.OldElectricityIndex
		var desc string
		if usage > 0 {
			desc = fmt.Sprintf("Tiêu thụ: %d kWh (Xem bảng chỉ số)", usage)
		} else {
			desc = "Định mức điện cố định"
		}
		lines = append(lines, invoiceLine{"Tiền điện", desc, invoice.ElectricityFee, false})
	}
	if invoice.WaterFee > 0 {
		usage := invoice.NewWaterIndex - invoice.OldWaterIndex
		var desc string
		if usage > 0 {
			desc = fmt.Sprintf("Tiêu thụ: %d m³ (Xem bảng chỉ số)", usage)
		} else {
			desc = "Định mức nước cố định"
		}
		lines = append(lines, invoiceLine{"Tiền nước", desc, invoice.WaterFee, false})
	}
	if invoice.WifiFee > 0 {
		lines = append(lines, invoiceLine{"Tiền mạng Wifi", "Trọn gói / phòng", invoice.WifiFee, false})
	}
	if invoice.ParkingFee > 0 {
		lines = append(lines, invoiceLine{"Tiền gửi xe", fmt.Sprintf("Gửi %d xe máy", invoice.VehicleCount), invoice.ParkingFee, false})
	}
	if invoice.ServiceFee > 0 {
		lines = append(lines, invoiceLine{"Phí dịch vụ chung", "Vệ sinh, rác, thang máy", invoice.ServiceFee, false})
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
		lines = append(lines, invoiceLine{"Giảm trừ khuyến mại", "Khấu trừ đặc biệt", invoice.Discount, true})
	}

	// Layout and height calculations
	yTable2 := 215.0
	var yRow float64
	if showUtilityTable {
		yRow = 278.0 + 32.0*float64(len(utilLines))
		yTable2 = yRow + 20.0
	}
	yStart := yTable2 + 58.0
	yEndTable2 := yStart + float64(len(lines))*48.0
	yTotal := yEndTable2 + 15.0
	neededHeight := int(yTotal + 75.0 + 40.0)

	// Create vertical ticket dynamically
	dc := gg.NewContext(600, neededHeight)
	
	// Soft elegant background
	dc.SetHexColor("#f8fafc")
	dc.Clear()

	// Outer card border shadow-like effect
	dc.SetHexColor("#cbd5e1")
	dc.DrawRoundedRectangle(24, 24, 552, float64(neededHeight-48), 16)
	dc.Fill()

	// Main White Card background
	dc.SetHexColor("#ffffff")
	dc.DrawRoundedRectangle(25, 25, 550, float64(neededHeight-50), 16)
	dc.Fill()

	// Title
	dc.SetHexColor("#1e1b4b")
	if err := loadFont(dc, "Roboto-Bold.ttf", 26); err == nil {
		dc.DrawStringAnchored("HÓA ĐƠN TIỀN NHÀ", 300, 70, 0.5, 0.5)
	}

	// Subtitle - Period
	dc.SetHexColor("#64748b")
	if err := loadFont(dc, "Roboto-Regular.ttf", 15); err == nil {
		dc.DrawStringAnchored(fmt.Sprintf("Kỳ hóa đơn: %s", invoice.Period), 300, 100, 0.5, 0.5)
	}

	// Metadata Box Background (Shrunk to show Room name only)
	dc.SetHexColor("#f1f5f9")
	dc.DrawRoundedRectangle(50, 130, 500, 50, 8)
	dc.Fill()

	// Metadata Text
	dc.SetHexColor("#0f172a")
	if err := loadFont(dc, "Roboto-Bold.ttf", 17); err == nil {
		dc.DrawStringAnchored(fmt.Sprintf("Phòng: %s", invoice.RoomName), 300, 155, 0.5, 0.5)
	}

	// RENDER TABLE 1: UTILITY STATEMENT (IF NEEDED)
	if showUtilityTable {
		// Table 1 Title
		dc.SetHexColor("#4f46e5")
		_ = loadFont(dc, "Roboto-Bold.ttf", 13)
		dc.DrawString("1. BẢNG KÊ CHỈ SỐ ĐIỆN / NƯỚC", 60, 220)

		// Table 1 Headers
		dc.SetHexColor("#64748b")
		_ = loadFont(dc, "Roboto-Bold.ttf", 10)
		dc.DrawString("DỊCH VỤ", 60, 242)
		dc.DrawStringAnchored("CHỈ SỐ CŨ", 220, 242, 0.5, 0)
		dc.DrawStringAnchored("CHỈ SỐ MỚI", 310, 242, 0.5, 0)
		dc.DrawStringAnchored("TIÊU THỤ", 400, 242, 0.5, 0)
		dc.DrawStringAnchored("ĐƠN GIÁ", 540, 242, 1.0, 0)

		// Separator Line
		dc.SetHexColor("#e2e8f0")
		dc.SetLineWidth(1)
		dc.DrawLine(50, 256, 550, 256)
		dc.Stroke()

		yRow = 278.0
		for _, ul := range utilLines {
			// Service Name
			dc.SetHexColor("#0f172a")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawString(ul.name, 60, yRow)

			// Indices and Details
			dc.SetHexColor("#334155")
			_ = loadFont(dc, "Roboto-Regular.ttf", 13)
			dc.DrawStringAnchored(fmt.Sprintf("%d", ul.oldIdx), 220, yRow, 0.5, 0)
			dc.DrawStringAnchored(fmt.Sprintf("%d", ul.newIdx), 310, yRow, 0.5, 0)
			dc.DrawStringAnchored(fmt.Sprintf("%d %s", ul.usage, ul.unit), 400, yRow, 0.5, 0)
			dc.DrawStringAnchored(formatCurrencyToVND(ul.unitPrice), 540, yRow, 1.0, 0)

			// Subtle Row separator
			dc.SetHexColor("#f1f5f9")
			dc.DrawLine(50, yRow+12, 550, yRow+12)
			dc.Stroke()

			yRow += 32.0
		}

		// Outer Separator for Table 1
		dc.SetHexColor("#cbd5e1")
		dc.DrawLine(50, yRow-10, 550, yRow-10)
		dc.Stroke()
	}

	// RENDER TABLE 2: SERVICE FEES STATEMENT
	// Table 2 Title
	dc.SetHexColor("#4f46e5")
	_ = loadFont(dc, "Roboto-Bold.ttf", 13)
	dc.DrawString("2. CHI TIẾT CÁC CHI PHÍ DỊCH VỤ", 60, yTable2)

	// Table 2 Headers
	dc.SetHexColor("#64748b")
	_ = loadFont(dc, "Roboto-Bold.ttf", 10)
	dc.DrawString("KHOẢN MỤC DỊCH VỤ", 60, yTable2+22)
	dc.DrawStringAnchored("MÔ TẢ CHI TIẾT", 320, yTable2+22, 0.5, 0)
	dc.DrawStringAnchored("THÀNH TIỀN", 540, yTable2+22, 1.0, 0)

	// Separator Under Headers
	dc.SetHexColor("#cbd5e1")
	dc.SetLineWidth(1)
	dc.DrawLine(50, yTable2+36, 550, yTable2+36)
	dc.Stroke()

	lineGap := 48.0
	for _, line := range lines {
		// Draw label
		dc.SetHexColor("#0f172a")
		_ = loadFont(dc, "Roboto-Bold.ttf", 13)
		dc.DrawString(line.label, 60, yStart)

		// Draw description below label
		dc.SetHexColor("#64748b")
		_ = loadFont(dc, "Roboto-Regular.ttf", 11)
		dc.DrawString(line.desc, 60, yStart+18)

		// Draw value aligned right
		if line.isDiscount {
			dc.SetHexColor("#dc2626") // Red for negative discount
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawStringAnchored(fmt.Sprintf("-%s", formatCurrencyToVND(line.value)), 540, yStart+10, 1.0, 0.5)
		} else {
			dc.SetHexColor("#0f172a")
			_ = loadFont(dc, "Roboto-Bold.ttf", 13)
			dc.DrawStringAnchored(formatCurrencyToVND(line.value), 540, yStart+10, 1.0, 0.5)
		}

		// Row Separator Line
		dc.SetHexColor("#f1f5f9")
		dc.SetLineWidth(1)
		dc.DrawLine(50, yStart+32, 550, yStart+32)
		dc.Stroke()

		yStart += lineGap
	}

	// Total Banner Card (Fixed bottom layout relative to rows)
	dc.SetHexColor("#e0e7ff") // Soft violet theme card
	dc.DrawRoundedRectangle(50, yTotal, 500, 75, 10)
	dc.Fill()

	// Total Label
	dc.SetHexColor("#3730a3") // Deep indigo
	_ = loadFont(dc, "Roboto-Bold.ttf", 16)
	dc.DrawString("TỔNG TIỀN CẦN THANH TOÁN", 70, yTotal+42)

	// Total Amount
	dc.SetHexColor("#4f46e5")
	_ = loadFont(dc, "Roboto-Bold.ttf", 24)
	dc.DrawStringAnchored(formatCurrencyToVND(invoice.TotalAmount), 530, yTotal+42, 1.0, 0.5)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"phongtro_hoadon_%s_%s.png\"", invoice.RoomName, invoice.Period))

	if err := png.Encode(w, dc.Image()); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode image", err)
	}
}
