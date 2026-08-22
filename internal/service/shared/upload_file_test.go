package shared

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	form, err := multipart.NewReader(body, writer.Boundary()).ReadForm(1024)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	return form.File["file"][0]
}

// Hồi quy: trước đây saveFile dùng os.Create nên upload CCCD/hợp đồng thất bại
// khi uploads/tenants chưa tồn tại.
func TestSaveUploadedFilesCreatesUploadDir(t *testing.T) {
	t.Chdir(t.TempDir())

	if _, err := os.Stat(TenantUploadDir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent before upload", TenantUploadDir)
	}

	header := newFileHeader(t, "cccd.jpg", []byte("fake-image"))

	paths, err := SaveUploadedFiles([]*multipart.FileHeader{header})
	if err != nil {
		t.Fatalf("processUploadedFiles: %v", err)
	}

	fileName, ok := strings.CutPrefix(paths, TenantFileURLPrefix)
	if !ok {
		t.Fatalf("unexpected stored path: %q", paths)
	}

	info, err := os.Stat(filepath.Join(TenantUploadDir, fileName))
	if err != nil {
		t.Fatalf("uploaded file missing: %v", err)
	}
	// CCCD và hợp đồng là dữ liệu cá nhân nhạy cảm nên chỉ owner được đọc.
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected file mode 0600, got %o", perm)
	}
}
