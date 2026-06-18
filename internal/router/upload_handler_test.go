package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// TestSignedUploadFileHandlerServesPlainAndSignedURLs verifies Zalo-compatible image access.
func TestSignedUploadFileHandlerServesPlainAndSignedURLs(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "invoice.png"), []byte("png"), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	const signingKey = "test-signing-key"
	publicPath := "/api/v1/uploads/zalo-invoices/invoice.png"
	values, err := security.SignPath(publicPath, time.Now().Add(time.Minute), signingKey)
	if err != nil {
		t.Fatalf("sign path: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/api/v1/uploads/zalo-invoices/*", signedUploadFileHandler(tmpDir, signingKey))

	req := httptest.NewRequest(http.MethodGet, publicPath+"?"+values.Encode(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("signed status = %d, want %d", rec.Code, http.StatusOK)
	}

	req = httptest.NewRequest(http.MethodGet, publicPath, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsigned status = %d, want %d", rec.Code, http.StatusOK)
	}

	req = httptest.NewRequest(http.MethodGet, publicPath+"?expires=1&sig=bad", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("invalid signed status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestSafeUploadFilePathRejectsTraversal covers path traversal in upload filename routing.
func TestSafeUploadFilePathRejectsTraversal(t *testing.T) {
	if _, ok := safeUploadFilePath("uploads/zalo-invoices", "../secret.png"); ok {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, ok := safeUploadFilePath("uploads/zalo-invoices", "nested/secret.png"); ok {
		t.Fatal("expected nested path to be rejected")
	}
}
