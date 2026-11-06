package apperr

import "errors"

type Kind uint8

const (
	KindInvalid Kind = iota + 1
	KindNotFound
	KindForbidden
	KindConflict
	KindUnauthorized
)

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && t.Message == e.Message
}

func (e *Error) Wrap(err error) *Error {
	c := *e
	c.Err = err
	return &c
}

func Invalid(message string) *Error      { return &Error{Kind: KindInvalid, Message: message} }
func NotFound(message string) *Error     { return &Error{Kind: KindNotFound, Message: message} }
func Forbidden(message string) *Error    { return &Error{Kind: KindForbidden, Message: message} }
func Conflict(message string) *Error     { return &Error{Kind: KindConflict, Message: message} }
func Unauthorized(message string) *Error { return &Error{Kind: KindUnauthorized, Message: message} }

func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
