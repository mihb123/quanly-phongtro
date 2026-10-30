package bankstatement

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/plugin"
	"github.com/mihb123/quanly-phongtro/pkg/pdftext"
	"github.com/mihb123/quanly-phongtro/pkg/zipcrypto"
)

const (
	PluginID      = "bank-statement"
	PaymentMethod = "BANK_STATEMENT"
	maxUploadSize = 25 << 20
)

const (
	StatusPaid            = "paid"
	StatusMatched         = "matched"
	StatusAlreadyRecorded = "already_recorded"
	StatusInvoicePaid     = "invoice_paid"
	StatusAmbiguous       = "ambiguous"
	StatusUnmatched       = "unmatched"
	StatusFailed          = "failed"
)

func init() {
	plugin.Register(&Plugin{})
}

type Plugin struct {
	host plugin.Host
}

const (
	SettingZipPassword  = "zip_password"
	SettingPreviewFirst = "preview_first"
	SettingRoundingDiff = "rounding_diff"
)

type invoiceSummary struct {
	ID          string  `json:"id"`
	RoomName    string  `json:"room_name"`
	HouseName   string  `json:"house_name"`
	Period      string  `json:"period"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
}

type importItem struct {
	Transaction
	Status     string           `json:"status"`
	Method     string           `json:"method,omitempty"`
	Message    string           `json:"message,omitempty"`
	Invoice    *invoiceSummary  `json:"invoice,omitempty"`
	Candidates []invoiceSummary `json:"candidates,omitempty"`
}

type importSummary struct {
	Credits         int   `json:"credits"`
	TotalCredit     int64 `json:"total_credit"`
	Paid            int   `json:"paid"`
	Matched         int   `json:"matched"`
	AlreadyRecorded int   `json:"already_recorded"`
	InvoicePaid     int   `json:"invoice_paid"`
	Ambiguous       int   `json:"ambiguous"`
	Unmatched       int   `json:"unmatched"`
	Failed          int   `json:"failed"`
}

type importResult struct {
	DryRun  bool          `json:"dry_run"`
	Files   []string      `json:"files"`
	Summary importSummary `json:"summary"`
	Items   []importItem  `json:"items"`
}

func (p *Plugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:             PluginID,
		Name:           "Đối soát sao kê ngân hàng",
		Version:        "1.0.0",
		Description:    "Tải file sao kê (.zip/.pdf) từ ngân hàng, tự khớp các khoản tiền vào với hóa đơn chưa thanh toán và đánh dấu đã thanh toán.",
		Author:         "quanly-phongtro",
		DefaultEnabled: true,
		Settings: []plugin.SettingField{
			{
				Key:         SettingPreviewFirst,
				Type:        plugin.SettingToggle,
				Group:       "Nhập sao kê",
				Label:       "Luôn xem trước khi cập nhật",
				Description: "Tải file lên chỉ hiện kết quả khớp; bạn chọn giao dịch rồi mới đánh dấu hóa đơn đã thanh toán.",
				Default:     true,
			},
			{
				Key:         SettingRoundingDiff,
				Type:        plugin.SettingSelect,
				Group:       "Khớp hóa đơn",
				Label:       "Chấp nhận chuyển lệch do làm tròn",
				Description: "Khách hay làm tròn số tiền. Khoản lệch chỉ được nhận khi nội dung chuyển khoản ghi tên khách hoặc số phòng và chỉ khớp đúng 1 hóa đơn.",
				Default:     "1000",
				Options: []plugin.SettingOption{
					{Value: "0", Label: "Không, phải đúng số tiền"},
					{Value: "1000", Label: "Lệch dưới 1.000 ₫"},
					{Value: "5000", Label: "Lệch dưới 5.000 ₫"},
					{Value: "10000", Label: "Lệch dưới 10.000 ₫"},
				},
			},
			{
				Key:         SettingZipPassword,
				Type:        plugin.SettingPassword,
				Group:       "Bảo mật",
				Label:       "Mật khẩu giải nén file zip",
				Description: "Tự dùng khi tải file zip mà không nhập mật khẩu. Lưu mã hóa trên máy chủ.",
				Placeholder: "Mật khẩu ngân hàng gửi kèm sao kê",
			},
		},
	}
}

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	return nil
}

func (p *Plugin) Routes(r chi.Router) {
	r.Post("/import", p.importStatement)
}

func (p *Plugin) importStatement(w http.ResponseWriter, r *http.Request) {
	managerID, ok := plugin.ManagerID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+(1<<20))
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		plugin.WriteError(w, http.StatusBadRequest, "File quá lớn hoặc dữ liệu tải lên không hợp lệ (tối đa 25MB)")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		plugin.WriteError(w, http.StatusBadRequest, "Vui lòng chọn file sao kê (.zip hoặc .pdf)")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		plugin.WriteError(w, http.StatusBadRequest, "Không đọc được file tải lên")
		return
	}

	ctx := r.Context()
	password := r.FormValue("password")
	remember := r.FormValue("remember_password") == "true"
	dryRun := r.FormValue("dry_run") == "true"
	config, err := p.host.Config().Get(ctx, managerID)
	if err != nil {
		log.Printf("bank-statement: load config for %s: %v", managerID, err)
		config = plugin.Values{}
	}
	usedSavedPassword := false
	if password == "" && config.String(SettingZipPassword) != "" {
		password = config.String(SettingZipPassword)
		usedSavedPassword = true
	}

	files, status, message := readStatementFiles(header.Filename, data, password, usedSavedPassword)
	if message != "" {
		plugin.WriteError(w, status, message)
		return
	}
	if remember && !usedSavedPassword && password != "" && bytes.HasPrefix(data, []byte("PK")) {
		if err := p.host.Config().Set(ctx, managerID, map[string]any{SettingZipPassword: password}); err != nil {
			log.Printf("bank-statement: save password for %s: %v", managerID, err)
		}
	}

	credits, err := parseCredits(files)
	if err != nil || len(credits) == 0 {
		plugin.WriteError(w, http.StatusUnprocessableEntity, "Không tìm thấy giao dịch ghi có nào trong file. Hiện hỗ trợ sao kê TPBank (phiếu báo có / sổ phụ tài khoản).")
		return
	}

	var selected map[string]bool
	if refs := r.MultipartForm.Value["references"]; len(refs) > 0 {
		selected = make(map[string]bool, len(refs))
		for _, ref := range refs {
			selected[ref] = true
		}
	}

	roundingDiff, _ := strconv.ParseInt(config.String(SettingRoundingDiff), 10, 64)
	result, err := p.reconcile(r, managerID, credits, dryRun, selected, roundingDiff)
	if err != nil {
		plugin.WriteError(w, http.StatusInternalServerError, "Không đối soát được sao kê, vui lòng thử lại")
		return
	}
	for _, f := range files {
		result.Files = append(result.Files, f.Name)
	}
	plugin.WriteJSON(w, http.StatusOK, result, "")
}

func readStatementFiles(name string, data []byte, password string, usedSavedPassword bool) ([]StatementFile, int, string) {
	type rawFile struct {
		name string
		data []byte
	}
	var raws []rawFile
	switch {
	case bytes.HasPrefix(data, []byte("PK")):
		entries, err := zipcrypto.Extract(data, password, zipcrypto.DefaultLimits)
		switch {
		case errors.Is(err, zipcrypto.ErrPasswordRequired):
			return nil, http.StatusBadRequest, "File zip có mật khẩu, vui lòng nhập mật khẩu giải nén"
		case errors.Is(err, zipcrypto.ErrWrongPassword) && usedSavedPassword:
			return nil, http.StatusBadRequest, "Mật khẩu đã lưu không mở được file này, vui lòng nhập mật khẩu khác"
		case errors.Is(err, zipcrypto.ErrWrongPassword):
			return nil, http.StatusBadRequest, "Sai mật khẩu giải nén file zip"
		case errors.Is(err, zipcrypto.ErrTooLarge):
			return nil, http.StatusBadRequest, "File zip giải nén ra quá lớn"
		case errors.Is(err, zipcrypto.ErrUnsupported):
			return nil, http.StatusBadRequest, "File zip dùng kiểu mã hóa chưa được hỗ trợ (chỉ hỗ trợ ZipCrypto)"
		case err != nil:
			return nil, http.StatusBadRequest, "File zip bị lỗi hoặc không đọc được"
		}
		for _, e := range entries {
			raws = append(raws, rawFile{name: path.Base(e.Name), data: e.Data})
		}
	case bytes.HasPrefix(bytes.TrimLeft(data[:min(len(data), 1024)], "\x00\r\n\t "), []byte("%PDF")):
		raws = append(raws, rawFile{name: path.Base(name), data: data})
	default:
		return nil, http.StatusBadRequest, "Chỉ hỗ trợ file .zip hoặc .pdf"
	}

	var files []StatementFile
	for _, raw := range raws {
		if !strings.EqualFold(path.Ext(raw.name), ".pdf") && !bytes.HasPrefix(raw.data, []byte("%PDF")) {
			continue
		}
		pages, err := pdftext.Extract(raw.data)
		if err != nil {
			log.Printf("bank-statement: skip %s: %v", raw.name, err)
			continue
		}
		files = append(files, StatementFile{Name: raw.name, Pages: pages})
	}
	if len(files) == 0 {
		return nil, http.StatusUnprocessableEntity, "Không tìm thấy file PDF sao kê đọc được"
	}
	return files, 0, ""
}

func (p *Plugin) reconcile(r *http.Request, managerID string, credits []Transaction, dryRun bool, selected map[string]bool, roundingDiff int64) (*importResult, error) {
	ctx := r.Context()
	invoices := p.host.Invoices()
	result := &importResult{DryRun: dryRun, Items: make([]importItem, 0, len(credits))}

	var fresh []Transaction
	recorded := map[string]string{}
	var periods []string
	seenPeriod := map[string]bool{}
	for _, tx := range credits {
		result.Summary.Credits++
		result.Summary.TotalCredit += tx.Amount
		if period := tx.Date.Format("2006-01"); !seenPeriod[period] {
			seenPeriod[period] = true
			periods = append(periods, period)
		}
		invoiceID, done, err := invoices.RecordedPayment(ctx, PluginID, tx.Account, tx.Reference)
		if err != nil {
			return nil, err
		}
		if done {
			recorded[tx.Reference] = invoiceID
			continue
		}
		fresh = append(fresh, tx)
	}

	outstanding, err := invoices.ListOutstanding(ctx, managerID, periods)
	if err != nil {
		return nil, err
	}
	matches := map[string]matchResult{}
	var leftover []Transaction
	for _, m := range matchTransactions(fresh, outstanding, roundingDiff) {
		matches[m.Transaction.Reference] = m
		if m.Invoice == nil && len(m.Candidates) == 0 {
			leftover = append(leftover, m.Transaction)
		}
	}

	paid, err := invoices.ListPaid(ctx, managerID, periods)
	if err != nil {
		return nil, err
	}
	paidByID := make(map[string]model.OutstandingInvoice, len(paid))
	for _, inv := range paid {
		paidByID[inv.ID] = inv
	}
	paidMatches := map[string]matchResult{}
	for _, m := range matchTransactions(leftover, paid, roundingDiff) {
		paidMatches[m.Transaction.Reference] = m
	}

	for _, tx := range credits {
		item := importItem{Transaction: tx}
		m, isFresh := matches[tx.Reference]
		invoiceID, isRecorded := recorded[tx.Reference]
		pm := paidMatches[tx.Reference]
		switch {
		case isRecorded || !isFresh:
			item.Status = StatusAlreadyRecorded
			item.Message = "Đã đối soát ở lần nhập trước, hóa đơn đã được đánh dấu thanh toán"
			if inv, ok := paidByID[invoiceID]; ok {
				item.Invoice = summarize(inv)
			}
		case m.Invoice != nil:
			item.Invoice = summarize(*m.Invoice)
			item.Method = m.Method
			item.Status = StatusMatched
			if !dryRun && (selected == nil || selected[tx.Reference]) {
				item.Status, item.Message = p.settle(r, managerID, tx, m)
			}
		case len(m.Candidates) > 0:
			item.Status = StatusAmbiguous
			item.Message = "Khớp số tiền với nhiều hóa đơn chưa thanh toán, cần xác nhận thủ công"
			for _, c := range m.Candidates {
				item.Candidates = append(item.Candidates, *summarize(c))
			}
		case pm.Invoice != nil:
			item.Invoice = summarize(*pm.Invoice)
			item.Method = pm.Method
			item.Status = StatusInvoicePaid
			item.Message = "Hóa đơn đã ở trạng thái đã thanh toán, không cập nhật lại"
		case len(pm.Candidates) > 0:
			item.Status = StatusInvoicePaid
			item.Message = "Khớp số tiền với nhiều hóa đơn đều đã thanh toán, bỏ qua"
			for _, c := range pm.Candidates {
				item.Candidates = append(item.Candidates, *summarize(c))
			}
		default:
			item.Status = StatusUnmatched
			item.Message = "Không có hóa đơn nào khớp số tiền"
		}
		countStatus(&result.Summary, item.Status)
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (p *Plugin) settle(r *http.Request, managerID string, tx Transaction, m matchResult) (string, string) {
	raw, _ := json.Marshal(tx)
	err := p.host.Invoices().MarkPaid(r.Context(), managerID, model.PluginPayment{
		Source:               PluginID,
		InvoiceID:            m.Invoice.ID,
		Amount:               int(tx.Amount),
		AccountRef:           tx.Account,
		TransactionReference: tx.Reference,
		MatchingMethod:       m.Method,
		PaymentMethod:        PaymentMethod,
		RawPayload:           string(raw),
	})
	switch {
	case err == nil:
		return StatusPaid, ""
	case errors.Is(err, model.ErrPluginPaymentRecorded):
		return StatusAlreadyRecorded, "Giao dịch đã được đối soát ở lần nhập trước"
	case errors.Is(err, model.ErrPluginInvoiceNotPayable):
		return StatusFailed, model.ErrPluginInvoiceNotPayable.Error()
	default:
		log.Printf("bank-statement: settle invoice %s with %s: %v", m.Invoice.ID, tx.Reference, err)
		return StatusFailed, "Không cập nhật được hóa đơn"
	}
}

func summarize(inv model.OutstandingInvoice) *invoiceSummary {
	return &invoiceSummary{ID: inv.ID, RoomName: inv.RoomName, HouseName: inv.HouseName, Period: inv.Period, TotalAmount: inv.TotalAmount, Status: inv.Status}
}

func countStatus(s *importSummary, status string) {
	switch status {
	case StatusPaid:
		s.Paid++
	case StatusMatched:
		s.Matched++
	case StatusAlreadyRecorded:
		s.AlreadyRecorded++
	case StatusInvoicePaid:
		s.InvoicePaid++
	case StatusAmbiguous:
		s.Ambiguous++
	case StatusUnmatched:
		s.Unmatched++
	case StatusFailed:
		s.Failed++
	}
}
