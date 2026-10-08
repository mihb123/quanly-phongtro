package zalobot

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingSecretToken = errors.New("zalobot: missing webhook secret token")
	ErrInvalidSecretToken = errors.New("zalobot: invalid webhook secret token")
	ErrInvalidImageURL    = errors.New("zalobot: invalid image url")
	ErrInvalidImage       = errors.New("zalobot: invalid image response")
)

const authErrorCode = -216

type APIError struct {
	Method      string
	StatusCode  int
	Code        int
	Description string
	Body        string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("zalobot: %s failed: status %d", e.Method, e.StatusCode)
	if e.Code != 0 {
		msg += fmt.Sprintf(", error %d", e.Code)
	}
	if e.Description != "" {
		return msg + ": " + e.Description
	}
	if e.Body != "" {
		return msg + ", body: " + e.Body
	}
	return msg
}

// IsAuthError checks if an error indicates that the Zalo bot token is invalid or expired.
// Zalo API typically returns error code -216 (invalid access token) for auth failures.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && (apiErr.Code == authErrorCode || apiErr.Code == 401 || apiErr.StatusCode == 401) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "-216") ||
		strings.Contains(msg, "invalid access token") ||
		strings.Contains(msg, "unauthorized")
}
