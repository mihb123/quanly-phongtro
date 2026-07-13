package logger_test

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

// captureLog redirects the standard logger output for the duration of fn and returns what was written.
func captureLog(fn func()) string {
	var builder strings.Builder
	originalFlags := log.Flags()
	log.SetOutput(&builder)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(originalFlags)
	}()
	fn()
	return builder.String()
}

// TestLoggerLevelsWithoutRequest exercises every level when no request is provided, with and without an error.
func TestLoggerLevelsWithoutRequest(t *testing.T) {
	levels := []struct {
		name string
		fn   func(*http.Request, int, string, error)
	}{
		{"debug", logger.Debug},
		{"info", logger.Info},
		{"warn", logger.Warn},
		{"error", logger.Error},
	}

	for _, level := range levels {
		t.Run(level.name+"_no_error", func(t *testing.T) {
			out := captureLog(func() {
				level.fn(nil, 200, "hello", nil)
			})
			if !strings.Contains(out, "status=200") || !strings.Contains(out, `msg="hello"`) {
				t.Errorf("unexpected log output: %q", out)
			}
		})

		t.Run(level.name+"_with_error", func(t *testing.T) {
			out := captureLog(func() {
				level.fn(nil, 500, "boom", errors.New("failure"))
			})
			if !strings.Contains(out, "err=failure") {
				t.Errorf("expected error in output: %q", out)
			}
		})
	}
}

// TestLoggerLevelsWithRequest exercises the request-aware branches, with and without an error.
func TestLoggerLevelsWithRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	request.RemoteAddr = "127.0.0.1:1234"

	out := captureLog(func() {
		logger.Info(request, 201, "created", nil)
	})
	if !strings.Contains(out, "method=POST") || !strings.Contains(out, "path=/api/v1/test") {
		t.Errorf("expected request details in output: %q", out)
	}

	out = captureLog(func() {
		logger.Error(request, 400, "bad", errors.New("invalid"))
	})
	if !strings.Contains(out, "remote=127.0.0.1:1234") || !strings.Contains(out, "err=invalid") {
		t.Errorf("expected request details and error in output: %q", out)
	}
}
