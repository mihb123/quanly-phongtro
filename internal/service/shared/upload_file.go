package shared

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// TenantUploadDir là nơi lưu CCCD của khách thuê và hợp đồng thuê phòng.
const TenantUploadDir = "uploads/tenants"

// OwnerUploadDir là nơi lưu CCCD chủ nhà và hợp đồng thuê nguyên căn, tách riêng khỏi hồ sơ khách thuê.
const OwnerUploadDir = "uploads/owners"

// TenantFileURLPrefix là prefix URL các file upload được phục vụ qua endpoint có xác thực.
const TenantFileURLPrefix = "/api/v1/tenant/files/"

var allowedUploadExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".pdf": true}

// SaveUploadedFiles lưu các file upload vào dir và trả về danh sách URL nội bộ, ngăn cách bởi dấu phẩy.
func SaveUploadedFiles(dir string, headers []*multipart.FileHeader) (string, error) {
	var paths []string
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return "", err
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if !allowedUploadExts[ext] {
			file.Close()
			return "", fmt.Errorf("unsupported file extension: %s", ext)
		}

		fileName := uuid.Must(uuid.NewV7()).String() + ext
		filePath := filepath.Join(dir, fileName)
		if err := saveFile(file, filePath); err != nil {
			file.Close()
			return "", err
		}
		file.Close()
		paths = append(paths, TenantFileURLPrefix+fileName)
	}
	return strings.Join(paths, ","), nil
}

func saveFile(file io.Reader, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create upload dir: %w", err)
	}
	// 0600 vì CCCD và hợp đồng là dữ liệu cá nhân nhạy cảm.
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("cannot create path %v", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return fmt.Errorf("cannot save file :%v", err)
	}
	return nil
}
