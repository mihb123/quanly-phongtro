package shared

import "errors"

// Shared validation errors returned across multiple service domains.
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidHouseID   = errors.New("invalid house id")
	ErrInvalidManagerID = errors.New("invalid manager id")
	ErrInvalidRoomID    = errors.New("invalid room id")
)
