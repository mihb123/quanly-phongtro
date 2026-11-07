// Package apperr cung cấp error mang thông tin tầng (Layer), tên hàm (Op) và vị trí
// file:line. Stack trace chỉ được chụp MỘT LẦN, tại mắt xích đầu tiên (lỗi gốc) của chuỗi.
package apperr

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

const (
	LayerRepo    = "repo"
	LayerService = "service"
	LayerHandler = "handler"
)

type Error struct {
	Layer  string
	Op     string
	Err    error
	Caller string // file:line nơi mắt xích này được tạo
	stack  []byte // chỉ khác nil ở lỗi gốc
}

func (e *Error) Error() string { return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Op, e.Err) }
func (e *Error) Unwrap() error { return e.Err }

// Wrap bọc err. Trả về nil nếu err == nil.
func Wrap(layer, op string, err error) error { return wrap(3, layer, op, err) }

// Wrapf như Wrap, thêm ngữ cảnh: "[layer:op] ctx: err".
func Wrapf(layer, op string, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return wrap(3, layer, op, fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
}

// New tạo lỗi gốc (không có cause).
func New(layer, op, msg string) error { return wrap(3, layer, op, errors.New(msg)) }

func Repo(op string, err error) error    { return wrap(3, LayerRepo, op, err) }
func Service(op string, err error) error { return wrap(3, LayerService, op, err) }

func wrap(skip int, layer, op string, err error) error {
	if err == nil {
		return nil
	}
	e := &Error{Layer: layer, Op: op, Err: err, Caller: caller(skip)}
	var inner *Error
	if !errors.As(err, &inner) { // lỗi gốc: chụp stack đúng một lần
		e.stack = debug.Stack()
	}
	return e
}

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/%s:%d", filepath.Base(filepath.Dir(file)), filepath.Base(file), line)
}

func root(err error) *Error {
	var last *Error
	for err != nil {
		var ae *Error
		if !errors.As(err, &ae) {
			break
		}
		last = ae
		err = ae.Err
	}
	return last
}

// FailedLayer: tầng sâu nhất trong chuỗi ("" nếu không có apperr).
func FailedLayer(err error) string {
	if r := root(err); r != nil {
		return r.Layer
	}
	return ""
}

// Origin: file:line của mắt xích sâu nhất.
func Origin(err error) string {
	if r := root(err); r != nil {
		return r.Caller
	}
	return ""
}

// Stack: stack trace chụp tại lỗi gốc (nil nếu không có).
func Stack(err error) []byte {
	if r := root(err); r != nil {
		return r.stack
	}
	return nil
}

// Message: thông điệp của lỗi gốc, không kèm tiền tố "[layer:op]". Dùng khi cần
// trả cho client câu chữ của một sentinel/lỗi nghiệp vụ đã được wrap (vd. 4xx).
// KHÔNG dùng cho lỗi 5xx: thông điệp gốc có thể chứa chi tiết nội bộ.
func Message(err error) string {
	if err == nil {
		return ""
	}
	if r := root(err); r != nil {
		return r.Err.Error()
	}
	return err.Error()
}
