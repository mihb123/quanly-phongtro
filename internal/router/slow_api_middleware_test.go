package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

func TestSlowAPILoggerLogsOnlySlowRequests(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "slow_api.log")
	slowLogger, err := logger.NewSlowAPILogger(logger.SlowAPIConfig{
		Threshold: 20 * time.Millisecond,
		FilePath:  base,
		MaxDays:   7,
	})
	if err != nil {
		t.Fatalf("new slow api logger: %v", err)
	}
	defer slowLogger.Close()

	r := chi.NewRouter()
	r.Use(slowAPILogger(slowLogger))
	r.Get("/api/v1/fast", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/api/v1/slow/{id}", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(40 * time.Millisecond)
		_, _ = w.Write([]byte("slow response"))
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/fast", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/slow/42?page=2", nil)
	req.Header.Set("User-Agent", "go-test")
	r.ServeHTTP(httptest.NewRecorder(), req)

	logPath := filepath.Join(dir, "slow_api-"+time.Now().Format("2006-01-02")+".log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read slow api log: %v", err)
	}

	var record struct {
		Context struct {
			Method       string  `json:"method"`
			URI          string  `json:"uri"`
			Route        string  `json:"route"`
			QueryParams  string  `json:"query_params"`
			DurationMs   float64 `json:"duration_ms"`
			StatusCode   int     `json:"status_code"`
			UserAgent    string  `json:"user_agent"`
			ResponseSize int     `json:"response_size"`
		} `json:"context"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("unmarshal log line: %v (%s)", err, data)
	}

	if record.Context.URI != "/api/v1/slow/42?page=2" {
		t.Fatalf("only the slow request must be logged, got %q", record.Context.URI)
	}
	if record.Context.Route != "/api/v1/slow/{id}" {
		t.Fatalf("route = %q, want route pattern", record.Context.Route)
	}
	if record.Context.QueryParams != "page=2" || record.Context.Method != http.MethodGet {
		t.Fatalf("unexpected context: %+v", record.Context)
	}
	if record.Context.DurationMs < 40 {
		t.Fatalf("duration_ms = %v, want >= 40", record.Context.DurationMs)
	}
	if record.Context.StatusCode != http.StatusOK || record.Context.ResponseSize != len("slow response") {
		t.Fatalf("unexpected status/size: %+v", record.Context)
	}
	if record.Context.UserAgent != "go-test" {
		t.Fatalf("user_agent = %q", record.Context.UserAgent)
	}
}

func TestSlowAPILoggerDisabledIsPassThrough(t *testing.T) {
	r := chi.NewRouter()
	r.Use(slowAPILogger(nil))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
