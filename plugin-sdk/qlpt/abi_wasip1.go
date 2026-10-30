//go:build wasip1

package qlpt

import "unsafe"

//go:wasmimport qlpt call
func hostCall(ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport qlpt result
func hostCopyResult(ptr unsafe.Pointer)

func transport(payload []byte) []byte {
	n := hostCall(unsafe.Pointer(unsafe.SliceData(payload)), uint32(len(payload)))
	out := make([]byte, n)
	if n > 0 {
		hostCopyResult(unsafe.Pointer(&out[0]))
	}
	return out
}
