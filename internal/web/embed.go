// Package web nhúng bản build tĩnh của frontend (Vite SPA) vào binary để API
// có thể phục vụ giao diện từ cùng một origin, tạo ra một file thực thi duy nhất.
package web

import (
	"embed"
	"io/fs"
)

// distFS chứa toàn bộ thư mục dist được sinh ra bởi `pnpm build` và copy vào đây
// trước khi `go build`. Tiền tố all: để nhúng cả các file ẩn (vd .gitkeep) nên
// build luôn biên dịch được kể cả khi frontend chưa được build.
//
//go:embed all:dist
var distFS embed.FS

// DistFS trả về hệ thống file của frontend đã nhúng, lấy gốc là thư mục dist.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
