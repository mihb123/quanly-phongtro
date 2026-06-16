package router

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// signedUploadFileHandler serves Zalo invoice images and validates signed URLs when present.
func signedUploadFileHandler(rootDir, signingKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := chi.URLParam(r, "*")
		filePath, ok := safeUploadFilePath(rootDir, fileName)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid file path")
			return
		}

		if hasSignedUploadQuery(r) {
			if err := security.VerifySignedPath(
				r.URL.Path,
				r.URL.Query().Get(security.SignedURLExpiresParam),
				r.URL.Query().Get(security.SignedURLSignatureParam),
				signingKey,
				time.Now(),
			); err != nil {
				writeError(w, http.StatusForbidden, "invalid signed url")
				return
			}
		}

		http.ServeFile(w, r, filePath)
	}
}

// hasSignedUploadQuery reports whether a request is attempting signed URL auth.
func hasSignedUploadQuery(r *http.Request) bool {
	query := r.URL.Query()
	return query.Get(security.SignedURLExpiresParam) != "" || query.Get(security.SignedURLSignatureParam) != ""
}

// safeUploadFilePath returns a path inside rootDir for a single upload filename.
func safeUploadFilePath(rootDir, fileName string) (string, bool) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || strings.ContainsAny(fileName, `/\`) {
		return "", false
	}
	if filepath.Clean(fileName) != fileName {
		return "", false
	}

	root := filepath.Clean(rootDir)
	filePath := filepath.Join(root, fileName)
	if filepath.Dir(filePath) != root {
		return "", false
	}
	return filePath, true
}
