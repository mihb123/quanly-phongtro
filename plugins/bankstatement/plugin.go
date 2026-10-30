package bankstatement

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path"
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

type savedSettings struct {
	ZipPassword string `json:"zip_password"`
}

type invoiceSummary struct {
	ID          string  `json:"id"`
	RoomName    string  `json:"room_name"`
	HouseName   string  `json:"house_name"`
	Period      string  `json:"period"`
	TotalAmount float64 `json:"total_amount"`
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
	}
}

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	return nil
}

func (p *Plugin) Routes(r chi.Router) {
	r.Get("/settings", p.getSettings)
	r.Delete("/settings/password", p.deletePassword)
	r.Post("/import", p.importStatement)
}

func (p *Plugin) getSettings(w http.ResponseWriter, r *http.Request) {
	managerID, ok := plugin.ManagerID(w, r)
	if !ok {
		return
	}
	var settings savedSettings
	if _, err := p.host.Storage().Load(r.Context(), managerID, &settings); err != nil {
		plugin.WriteError(w, http.StatusInternalServerError, "Không đọc được cấu hình plugin")
		return
	}
	plugin.WriteJSON(w, http.StatusOK, map[string]bool{"has_saved_password": settings.ZipPassword != ""}, "")
}

func (p *Plugin) deletePassword(w http.ResponseWriter, r *http.Request) {
	managerID, ok := plugin.ManagerID(w, r)
	if !ok {
		return
	}
	if err := p.host.Storage().Clear(r.Context(), managerID); err != nil {
		plugin.WriteError(w, http.StatusInternalServerError, "Không xóa được mật khẩu đã lưu")
		return
	}
	plugin.WriteJSON(w, http.StatusOK, map[string]bool{"has_saved_password": false}, "")
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
	usedSavedPassword := false
	if password == "" {
		var settings savedSettings
		if _, err := p.host.Storage().Load(ctx, managerID, &settings); err == nil && settings.ZipPassword != "" {
			password = settings.ZipPassword
			usedSavedPassword = true
		}
	}

	files, status, message := readStatementFiles(header.Filename, data, password, usedSavedPassword)
	if message != "" {
		plugin.WriteError(w, status, message)
		return
	}
	if remember && !usedSavedPassword && password != "" && bytes.HasPrefix(data, []byte("PK")) {
		if err := p.host.Storage().Save(ctx, managerID, savedSettings{ZipPassword: password}); err != nil {
			log.Printf("bank-statement: save password for %s: %v", managerID, err)
		}
	}

	credits, err := parseCredits(files)
	if err != nil || len(credits) == 0 {
		plugin.WriteError(w, http.StatusUnprocessableEntity, "Không tìm thấy giao dịch ghi có nào trong file. Hiện hỗ trợ sao kê TPBank (phiếu báo có / sổ phụ tài khoản).")
		return
	}

	result, err := p.reconcile(r, managerID, credits, dryRun)
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

func (p *Plugin) reconcile(r *http.Request, managerID string, credits []Transaction, dryRun bool) (*importResult, error) {
	ctx := r.Context()
	invoices := p.host.Invoices()
	result := &importResult{DryRun: dryRun, Items: make([]importItem, 0, len(credits))}

	var fresh []Transaction
	recorded := map[string]bool{}
	for _, tx := range credits {
		result.Summary.Credits++
		result.Summary.TotalCredit += tx.Amount
		done, err := invoices.PaymentRecorded(ctx, PluginID, tx.Account, tx.Reference)
		if err != nil {
			return nil, err
		}
		if done {
			recorded[tx.Reference] = true
			continue
		}
		fresh = append(fresh, tx)
	}

	outstanding, err := invoices.ListOutstanding(ctx, managerID)
	if err != nil {
		return nil, err
	}
	matches := map[string]matchResult{}
	for _, m := range matchTransactions(fresh, outstanding) {
		matches[m.Transaction.Reference] = m
	}

	for _, tx := range credits {
		item := importItem{Transaction: tx}
		m, isFresh := matches[tx.Reference]
		switch {
		case recorded[tx.Reference] || !isFresh:
			item.Status = StatusAlreadyRecorded
			item.Message = "Giao dịch đã được đối soát ở lần nhập trước"
		case m.Invoice != nil:
			item.Invoice = summarize(*m.Invoice)
			item.Method = m.Method
			item.Status = StatusMatched
			if !dryRun {
				item.Status, item.Message = p.settle(r, managerID, tx, m)
			}
		case len(m.Candidates) > 0:
			item.Status = StatusAmbiguous
			item.Message = "Có nhiều hóa đơn cùng số tiền, cần xác nhận thủ công"
			for _, c := range m.Candidates {
				item.Candidates = append(item.Candidates, *summarize(c))
			}
		default:
			item.Status = StatusUnmatched
			item.Message = "Không có hóa đơn chưa thanh toán nào khớp số tiền"
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
	return &invoiceSummary{ID: inv.ID, RoomName: inv.RoomName, HouseName: inv.HouseName, Period: inv.Period, TotalAmount: inv.TotalAmount}
}

func countStatus(s *importSummary, status string) {
	switch status {
	case StatusPaid:
		s.Paid++
	case StatusMatched:
		s.Matched++
	case StatusAlreadyRecorded:
		s.AlreadyRecorded++
	case StatusAmbiguous:
		s.Ambiguous++
	case StatusUnmatched:
		s.Unmatched++
	case StatusFailed:
		s.Failed++
	}
}
