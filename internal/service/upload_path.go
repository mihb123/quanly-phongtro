package service

import (
	"path/filepath"
	"strings"
)

// uploadFileName extracts a single safe filename from a stored or requested upload path.
func uploadFileName(requestPath string) (string, bool) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return "", false
	}

	candidate := strings.ReplaceAll(requestPath, `\`, "/")
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

// uploadFilePath joins a safe upload filename under the expected upload root.
func uploadFilePath(rootDir, fileName string) (string, bool) {
	if _, ok := uploadFileName(fileName); !ok {
		return "", false
	}

	root := filepath.Clean(rootDir)
	filePath := filepath.Join(root, fileName)
	if filepath.Dir(filePath) != root {
		return "", false
	}

	return filePath, true
}
