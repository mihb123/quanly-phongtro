package logger

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSlowAPILoggerDisabledWhenThresholdNotPositive(t *testing.T) {
	slowLogger, err := NewSlowAPILogger(SlowAPIConfig{Threshold: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slowLogger != nil {
		t.Fatalf("expected nil logger when threshold is 0")
	}

	// Logger nil vẫn phải an toàn khi middleware gọi tới.
	if slowLogger.IsSlow(time.Second) {
		t.Fatalf("nil logger must never report slow")
	}
	slowLogger.Log(SlowAPIEntry{})
	if err := slowLogger.Close(); err != nil {
		t.Fatalf("close nil logger: %v", err)
	}
}

func TestSlowAPILoggerWritesJSONLine(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "slow_api.log")
	slowLogger, err := NewSlowAPILogger(SlowAPIConfig{Threshold: time.Second, FilePath: base, MaxDays: 7})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer slowLogger.Close()

	now := time.Now()
	slowLogger.Log(SlowAPIEntry{
		Time:         now,
		Method:       http.MethodGet,
		URI:          "/api/v1/invoice?page=1",
		Path:         "/api/v1/invoice",
		Route:        "/api/v1/invoice/",
		QueryParams:  "page=1",
		StatusCode:   200,
		Duration:     4067140 * time.Microsecond,
		ClientIP:     "113.161.50.20",
		UserAgent:    "Mozilla/5.0",
		RequestSize:  12,
		ResponseSize: 3456,
	})

	data, err := os.ReadFile(filepath.Join(dir, "slow_api-"+now.Format("2006-01-02")+".log"))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	var record struct {
		Message   string `json:"message"`
		Channel   string `json:"channel"`
		LevelName string `json:"level_name"`
		Context   struct {
			Method     string  `json:"method"`
			URI        string  `json:"uri"`
			Route      string  `json:"route"`
			DurationMs float64 `json:"duration_ms"`
			StatusCode int     `json:"status_code"`
			ClientIP   string  `json:"client_ip"`
		} `json:"context"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("unmarshal log line: %v (%s)", err, data)
	}

	if record.Message != slowAPIMessage || record.Channel != slowAPIChannel || record.LevelName != slowAPILevelName {
		t.Fatalf("unexpected envelope: %+v", record)
	}
	if record.Context.DurationMs != 4067.14 {
		t.Fatalf("duration_ms = %v, want 4067.14", record.Context.DurationMs)
	}
	if record.Context.Route != "/api/v1/invoice/" || record.Context.URI != "/api/v1/invoice?page=1" {
		t.Fatalf("unexpected context: %+v", record.Context)
	}
	if record.Context.ClientIP != "113.161.50.20" || record.Context.StatusCode != 200 {
		t.Fatalf("unexpected context: %+v", record.Context)
	}
}

func TestSlowAPILoggerRotatesAndPurgesOldFiles(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "slow_api.log")
	slowLogger, err := NewSlowAPILogger(SlowAPIConfig{Threshold: time.Second, FilePath: base, MaxDays: 2})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer slowLogger.Close()

	day := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		slowLogger.Log(SlowAPIEntry{Time: day.AddDate(0, 0, i), Method: http.MethodGet, Path: "/health", Duration: 2 * time.Second})
	}

	files, err := filepath.Glob(filepath.Join(dir, "slow_api-*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 retained files, got %d (%v)", len(files), files)
	}
	if _, err := os.Stat(filepath.Join(dir, "slow_api-2026-09-16.log")); !os.IsNotExist(err) {
		t.Fatalf("oldest log file should have been purged, err = %v", err)
	}
}
