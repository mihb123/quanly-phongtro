//go:build !wasip1

package qlpt

func transport([]byte) []byte {
	panic("qlpt: backend plugin chỉ chạy trong host; build bằng GOOS=wasip1 GOARCH=wasm")
}
