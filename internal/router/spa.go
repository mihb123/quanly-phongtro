package router

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaFileServer phục vụ file tĩnh từ frontend đã nhúng. Với đường dẫn không tồn
// tại (route phía client như /houses/123), nó trả về index.html để SPA tự định
// tuyến; với đường dẫn /api/* không khớp route nào, nó trả lỗi JSON 404 để giữ
// đúng ngữ nghĩa API thay vì trả về trang HTML.
func spaFileServer(distFS fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(distFS))
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "resource not found")
			return
		}

		requestedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestedPath == "" {
			requestedPath = "index.html"
		}
		if _, err := fs.Stat(distFS, requestedPath); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}
}
