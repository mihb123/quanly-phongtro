package shared

import (
	"path/filepath"
	"strings"
)

// UploadFileName extracts a single safe filename from a stored or requested upload path.
func UploadFileName(requestPath string) (string, bool) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return "", false
	}

	requestPath = strings.ReplaceAll(requestPath, `\`, `/`)

	candidate := requestPath
	for _, prefix := range []string{
		"/api/v1/tenant/files/",
		"/api/v1/uploads/transactions/",
		"/uploads/transactions/",
	} {
		if strings.HasPrefix(candidate, prefix) {
			candidate = strings.TrimPrefix(candidate, prefix)
			break
		}
	}

	if candidate == "" || candidate == "." || candidate == "/" {
		return "", false
	}
	if strings.Contains(candidate, "/") || strings.Contains(candidate, "..") {
		return "", false
	}
	if filepath.Clean(candidate) != candidate {
		return "", false
	}

	return candidate, true
}

// UploadFilePath joins a safe upload filename under the expected upload root.
func UploadFilePath(rootDir, fileName string) (string, bool) {
	if _, ok := UploadFileName(fileName); !ok {
		return "", false
	}

	root := filepath.Clean(rootDir)
	filePath := filepath.Join(root, fileName)
	if filepath.Dir(filePath) != root {
		return "", false
	}

	return filePath, true
}
