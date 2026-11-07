package shared

import "errors"

// Shared validation errors returned across multiple service domains.
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidHouseID   = errors.New("invalid house id")
	ErrInvalidManagerID = errors.New("invalid manager id")
	ErrInvalidRoomID    = errors.New("invalid room id")
)

// invalidError là lỗi dữ liệu đầu vào có thông điệp dành cho người dùng.
type invalidError struct {
	msg   string
	cause error
}

func (e *invalidError) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}
func (e *invalidError) Unwrap() error         { return e.cause }
func (e *invalidError) Is(target error) bool  { return target == ErrInvalidInput }
func (e *invalidError) PublicMessage() string { return e.msg }

// Invalid tạo lỗi đầu vào với thông điệp hiển thị cho client (HTTP 400).
// errors.Is(err, ErrInvalidInput) == true; client nhận đúng msg.
func Invalid(msg string) error { return &invalidError{msg: msg} }

// InvalidCause như Invalid nhưng giữ lỗi gốc cho log (client chỉ thấy msg).
func InvalidCause(msg string, cause error) error { return &invalidError{msg: msg, cause: cause} }
