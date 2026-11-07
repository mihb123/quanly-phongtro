package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

//go:noinline
func repoQueryFails() error {
	return apperr.Repo("Query", errors.New("dial tcp: connection refused"))
}

func serviceListFails() error { return apperr.Service("ListHouse", repoQueryFails()) }

func newLoggedRouter(t *testing.T) (http.Handler, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	r := chi.NewRouter()
	r.Use(loggingMiddleware(zap.New(core), []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}))
	r.Use(recoverMiddleware)
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) { httpx.WriteJSON(w, http.StatusOK, nil, "ok") })
	r.Get("/empty", func(http.ResponseWriter, *http.Request) {})
	r.Get("/notfound", func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, apperr.Service("GetHouse", apperr.Repo("GetByID", model.ErrHouseNotFound)))
	})
	r.Get("/db", func(w http.ResponseWriter, r *http.Request) { httpx.Fail(w, r, serviceListFails()) })
	r.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("something went wrong") })
	r.Get("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		logging.SetUserID(r.Context(), "u-1")
		if logging.TraceID(r.Context()) == "" || !hasIP(r) {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return r, logs
}

func do(h http.Handler, path string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "127.0.0.1:5555"
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func onlyEntry(t *testing.T, logs *observer.ObservedLogs) observer.LoggedEntry {
	t.Helper()
	all := logs.TakeAll()
	if len(all) != 1 {
		t.Fatalf("want exactly 1 log line, got %d: %+v", len(all), all)
	}
	return all[0]
}

func TestLoggingOK(t *testing.T) {
	h, logs := newLoggedRouter(t)
	rec := do(h, "/ok", map[string]string{"CF-Connecting-IP": "113.161.50.20"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	e := onlyEntry(t, logs)
	m := e.ContextMap()
	if e.Level != zapcore.InfoLevel || e.Message != "request completed" {
		t.Fatalf("got %s %q", e.Level, e.Message)
	}
	if m["status"] != int64(200) || m["path"] != "/ok" || m["method"] != "GET" || m["ip"] != "113.161.50.20" || m["route"] != "/ok" {
		t.Fatalf("fields: %+v", m)
	}
	if _, has := m["error"]; has {
		t.Fatal("no error field expected")
	}
}

func TestLoggingEmptyHandlerIs200(t *testing.T) {
	h, logs := newLoggedRouter(t)
	do(h, "/empty", nil)
	if m := onlyEntry(t, logs).ContextMap(); m["status"] != int64(200) {
		t.Fatalf("status %v", m["status"])
	}
}

func TestLoggingSentinel404(t *testing.T) {
	h, logs := newLoggedRouter(t)
	rec := do(h, "/notfound", nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), model.ErrHouseNotFound.Error()) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "[repo:") {
		t.Fatalf("layer prefix leaked: %s", rec.Body)
	}
	e := onlyEntry(t, logs)
	if e.Level != zapcore.WarnLevel || e.Message != "request rejected" {
		t.Fatalf("got %s %q", e.Level, e.Message)
	}
	if _, has := e.ContextMap()["stack"]; has {
		t.Fatal("4xx must not carry a stack")
	}
}

func TestLoggingDBError500(t *testing.T) {
	h, logs := newLoggedRouter(t)
	rec := do(h, "/db", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("internal error leaked: %s", rec.Body)
	}
	e := onlyEntry(t, logs)
	m := e.ContextMap()
	if e.Level != zapcore.ErrorLevel || e.Message != "request failed" {
		t.Fatalf("got %s %q", e.Level, e.Message)
	}
	if m["failed_layer"] != "repo" {
		t.Fatalf("failed_layer %v", m["failed_layer"])
	}
	if errStr, _ := m["error"].(string); !strings.Contains(errStr, "[service:ListHouse] [repo:Query]") {
		t.Fatalf("error %q", errStr)
	}
	if origin, _ := m["origin"].(string); !strings.HasPrefix(origin, "router/middleware_test.go:") {
		t.Fatalf("origin %q", origin)
	}
	if stack, _ := m["stack"].(string); !strings.Contains(stack, "repoQueryFails") {
		t.Fatalf("stack must point at the root error, got:\n%s", stack)
	}
}

func TestLoggingPanic(t *testing.T) {
	h, logs := newLoggedRouter(t)
	rec := do(h, "/panic", nil)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "something went wrong") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	e := onlyEntry(t, logs)
	m := e.ContextMap()
	if e.Level != zapcore.ErrorLevel {
		t.Fatalf("level %s", e.Level)
	}
	if !strings.Contains(m["error"].(string), "something went wrong") {
		t.Fatalf("error %v", m["error"])
	}
	if stack, _ := m["stack"].(string); !strings.Contains(stack, "middleware_test.go") {
		t.Fatalf("stack must be taken at the panic site:\n%s", stack)
	}
}

func TestLoggingTraceIDAndUser(t *testing.T) {
	h, logs := newLoggedRouter(t)

	rec := do(h, "/user/1", nil)
	generated := rec.Header().Get(TraceIDHeader)
	if rec.Code != http.StatusNoContent || generated == "" {
		t.Fatalf("got %d, trace %q", rec.Code, generated)
	}
	m := onlyEntry(t, logs).ContextMap()
	if m["trace_id"] != generated || m["user_id"] != "u-1" {
		t.Fatalf("fields %+v", m)
	}

	rec = do(h, "/ok", map[string]string{TraceIDHeader: "client-trace-1"})
	if rec.Header().Get(TraceIDHeader) != "client-trace-1" {
		t.Fatal("client trace id must be reused")
	}
	if onlyEntry(t, logs).ContextMap()["trace_id"] != "client-trace-1" {
		t.Fatal("log must carry client trace id")
	}

	rec = do(h, "/ok", map[string]string{TraceIDHeader: "bad\nvalue"})
	if got := rec.Header().Get(TraceIDHeader); got == "bad\nvalue" || got == "" {
		t.Fatalf("invalid trace id must be replaced, got %q", got)
	}
	logs.TakeAll()
}

func hasIP(r *http.Request) bool {
	ip, ok := security.ClientIPFromContext(r.Context())
	return ok && ip != ""
}

func TestLoggingWarningsAndAnnotations(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	r := chi.NewRouter()
	r.Use(loggingMiddleware(zap.New(core), nil))
	r.Post("/hook", func(w http.ResponseWriter, r *http.Request) {
		logging.Annotate(r.Context(), zap.String("zalo_event", "message.text.received"))
		logging.RecordWarning(r.Context(), "send message", errors.New("zalo 500"))
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	e := onlyEntry(t, logs)
	m := e.ContextMap()
	if e.Level != zapcore.WarnLevel || e.Message != "request completed with warnings" {
		t.Fatalf("got %s %q", e.Level, e.Message)
	}
	if m["zalo_event"] != "message.text.received" {
		t.Fatalf("annotation missing: %+v", m)
	}
	if w, _ := m["warnings"].([]interface{}); len(w) != 1 || w[0] != "send message: zalo 500" {
		t.Fatalf("warnings %#v", m["warnings"])
	}
}

func TestLoggingNotedErrorOn200IsWarn(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	r := chi.NewRouter()
	r.Use(loggingMiddleware(zap.New(core), nil))
	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		httpx.NoteError(r, apperr.Service("HandleWebhook", errors.New("bad secret")))
		w.WriteHeader(http.StatusOK)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/webhook", nil))
	e := onlyEntry(t, logs)
	if e.Level != zapcore.WarnLevel || !strings.Contains(e.ContextMap()["error"].(string), "bad secret") {
		t.Fatalf("got %s %+v", e.Level, e.ContextMap())
	}
	if _, has := e.ContextMap()["stack"]; has {
		t.Fatal("non-5xx must not carry a stack")
	}
}
