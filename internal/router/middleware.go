package router

import (
	"fmt"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				err, ok := rec.(error)
				if !ok {
					err = fmt.Errorf("panic: %v", rec)
				}

				logger.Error(r, http.StatusInternalServerError, "panic recovered", err)
				// Chi tiết panic chỉ ghi log, client nhận thông điệp chung.
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
