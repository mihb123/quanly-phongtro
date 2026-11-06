package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/apperr"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

const (
	msgInternal   = "Có lỗi xảy ra, vui lòng thử lại"
	msgValidation = "Dữ liệu không hợp lệ"
)

type validationBody struct {
	Status  int               `json:"status"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func ValidationFailed(w http.ResponseWriter, errs *FieldErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(validationBody{
		Status:  http.StatusUnprocessableEntity,
		Message: errs.First(msgValidation),
		Errors:  errs.Map(),
	})
}

func Fail(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperr.As(err); ok {
		WriteError(w, statusOf(e.Kind), e.Message)
		return
	}
	if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
		return
	}
	logger.Error(r, http.StatusInternalServerError, "request failed", err)
	WriteError(w, http.StatusInternalServerError, msgInternal)
}

func statusOf(kind apperr.Kind) int {
	switch kind {
	case apperr.KindInvalid:
		return http.StatusUnprocessableEntity
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
