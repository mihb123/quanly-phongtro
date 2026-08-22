package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverMiddleware(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		recoverMiddleware(handler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Panic is recovered", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		recoverMiddleware(handler).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}

		// Chi tiết panic không được rò rỉ ra response, chỉ nằm trong log.
		body := rec.Body.String()
		if strings.Contains(body, "something went wrong") {
			t.Errorf("panic detail leaked to client: %s", body)
		}
		if !strings.Contains(body, "internal server error") {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
