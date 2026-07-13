package shared_test

import (
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/service/shared"
)

// TestDecodeAES256Key covers valid hex keys, raw 32-byte fallbacks, and rejection cases.
func TestDecodeAES256Key(t *testing.T) {
	validHex := hex.EncodeToString(make([]byte, 32))

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{"valid hex 32 bytes", validHex, false, 32},
		{"raw 32-byte non-hex string", strings.Repeat("z", 32), false, 32},
		{"invalid hex wrong length", "zzzz", true, 0},
		{"valid hex wrong length", hex.EncodeToString(make([]byte, 16)), true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, err := shared.DecodeAES256Key(tc.input, "TEST_KEY")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got key %v", key)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(key) != tc.wantLen {
				t.Errorf("expected key length %d, got %d", tc.wantLen, len(key))
			}
		})
	}
}

// TestDecodeEncryptionKey ensures the Zalo wrapper delegates to the AES-256 decoder.
func TestDecodeEncryptionKey(t *testing.T) {
	key, err := shared.DecodeEncryptionKey(hex.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}

	if _, err := shared.DecodeEncryptionKey("short"); err == nil {
		t.Error("expected error for invalid key")
	}
}

// TestUploadFileName covers prefix stripping, sanitization, and traversal rejection.
func TestUploadFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantOK   bool
	}{
		{"empty", "", "", false},
		{"whitespace", "   ", "", false},
		{"plain filename", "photo.jpg", "photo.jpg", true},
		{"tenant files prefix", "/api/v1/tenant/files/photo.jpg", "photo.jpg", true},
		{"uploads api prefix", "/api/v1/uploads/transactions/photo.jpg", "photo.jpg", true},
		{"uploads plain prefix", "/uploads/transactions/photo.jpg", "photo.jpg", true},
		{"backslash prefix normalized", `\uploads\transactions\photo.jpg`, "photo.jpg", true},
		{"backslash traversal rejected", `\..\secret`, "", false},
		{"dot", ".", "", false},
		{"slash", "/", "", false},
		{"nested path", "sub/photo.jpg", "", false},
		{"traversal", "..", "", false},
		{"traversal file", "../secret", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := shared.UploadFileName(tc.input)
			if ok != tc.wantOK || name != tc.wantName {
				t.Errorf("UploadFileName(%q) = (%q, %v), want (%q, %v)", tc.input, name, ok, tc.wantName, tc.wantOK)
			}
		})
	}
}

// TestUploadFilePath ensures a safe filename resolves under the root and invalid names are rejected.
func TestUploadFilePath(t *testing.T) {
	root := "/var/data/uploads"

	path, ok := shared.UploadFilePath(root, "photo.jpg")
	if !ok {
		t.Fatal("expected valid path")
	}
	if path != filepath.Join(root, "photo.jpg") {
		t.Errorf("unexpected path: %q", path)
	}

	if _, ok := shared.UploadFilePath(root, "../escape"); ok {
		t.Error("expected traversal filename to be rejected")
	}
}
