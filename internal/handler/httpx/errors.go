package httpx

import (
	"errors"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/model"
	authsvc "github.com/mihb123/quanly-phongtro/internal/service/auth"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
)

// InternalErrorMessage là thông điệp duy nhất client nhận cho lỗi không lường trước.
const InternalErrorMessage = "internal server error"

type errRule struct {
	target  error
	status  int
	message string // "" = dùng câu chữ của lỗi gốc (apperr.Message)
}

// errRules: sentinel -> HTTP status + message an toàn. Thêm sentinel mới = thêm 1 dòng.
// Thứ tự quan trọng: rule đầu tiên khớp được dùng.
var errRules = []errRule{
	// 400: dữ liệu đầu vào / trạng thái không hợp lệ.
	{sharedsvc.ErrInvalidInput, http.StatusBadRequest, ""},
	{sharedsvc.ErrInvalidHouseID, http.StatusBadRequest, ""},
	{sharedsvc.ErrInvalidManagerID, http.StatusBadRequest, ""},
	{sharedsvc.ErrInvalidRoomID, http.StatusBadRequest, ""},
	{model.ErrHouseCostInvalidID, http.StatusBadRequest, ""},
	{model.ErrMaxTenans, http.StatusBadRequest, ""},
	{model.ErrInvalidElectricityIndex, http.StatusBadRequest, ""},
	{model.ErrInvalidWaterIndex, http.StatusBadRequest, ""},
	{model.ErrInvoiceAlreadyPaid, http.StatusBadRequest, ""},
	{model.ErrInvoiceAlreadyUnpaid, http.StatusBadRequest, ""},
	{model.ErrPaidInvoiceImmutable, http.StatusBadRequest, ""},
	{model.ErrPaidInvoiceDelete, http.StatusBadRequest, ""},
	{model.ErrPluginInvoiceNotPayable, http.StatusBadRequest, ""},
	{paymentsvc.ErrPaymentProviderNotFound, http.StatusBadRequest, ""},
	{paymentsvc.ErrPaymentWebhookInvalid, http.StatusBadRequest, ""},

	// 401 / 403.
	{authsvc.ErrInvalidCredentials, http.StatusUnauthorized, ""},
	{model.ErrUnauthorized, http.StatusForbidden, ""},
	{model.ErrHouseCostForbidden, http.StatusForbidden, ""},
	{model.ErrPluginDisabled, http.StatusForbidden, ""},

	// 404.
	{model.ErrNotFound, http.StatusNotFound, ""},
	{model.ErrHouseNotFound, http.StatusNotFound, ""},
	{model.ErrRoomNotFound, http.StatusNotFound, ""},
	{model.ErrTenantNotFound, http.StatusNotFound, ""},
	{model.ErrInvoiceNotFound, http.StatusNotFound, ""},
	{model.ErrHouseCostNotFound, http.StatusNotFound, ""},
	{model.ErrPendingInvoiceUpdateNotFound, http.StatusNotFound, ""},
	{model.ErrPluginNotFound, http.StatusNotFound, ""},
	{paymentsvc.ErrPaymentCredentialsNotFound, http.StatusNotFound, ""},

	// 409.
	{model.ErrAlreadyExists, http.StatusConflict, ""},
	{model.ErrHouseCodeExists, http.StatusConflict, ""},
	{model.ErrPhoneAlreadyExists, http.StatusConflict, ""},
	{model.ErrDuplicateInvoice, http.StatusConflict, ""},
	{model.ErrHouseCostAlreadyExists, http.StatusConflict, ""},
	{model.ErrPluginPaymentRecorded, http.StatusConflict, ""},
	{authsvc.ErrEmailAlreadyExists, http.StatusConflict, ""},
}

// StatusOf map lỗi -> (status, message an toàn cho client). Lỗi không khớp rule nào = 500.
func StatusOf(err error) (int, string) {
	for _, r := range errRules {
		if errors.Is(err, r.target) {
			if r.message != "" {
				return r.status, r.message
			}
			return r.status, PublicMessage(err)
		}
	}
	return http.StatusInternalServerError, InternalErrorMessage
}

// publicMessager: lỗi tự mang câu chữ an toàn cho client (vd. sharedsvc.Invalid*).
type publicMessager interface{ PublicMessage() string }

// PublicMessage: câu chữ cho client của một lỗi 4xx đã biết — PublicMessage() nếu có,
// không thì thông điệp lỗi gốc bỏ tiền tố "[layer:op]".
func PublicMessage(err error) string {
	var pm publicMessager
	if errors.As(err, &pm) {
		return pm.PublicMessage()
	}
	return apperr.Message(err)
}

// Fail ghi err vào request state (để middleware log đúng một dòng) và trả JSON lỗi an toàn.
// Handler gọi `httpx.Fail(w, r, err); return` thay cho log + WriteError + err.Error().
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := StatusOf(err)
	logging.RecordError(r.Context(), err)
	WriteError(w, status, msg)
}

// FailWith như Fail nhưng dùng status/message do handler chọn (nhánh 4xx có câu chữ riêng).
func FailWith(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	logging.RecordError(r.Context(), err)
	WriteError(w, status, message)
}

// NoteError gắn lỗi vào dòng log của request mà không đổi response
// (vd. webhook luôn trả 200 để nhà cung cấp không gửi lại).
func NoteError(r *http.Request, err error) {
	logging.RecordError(r.Context(), err)
}
