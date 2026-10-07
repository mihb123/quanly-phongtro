package sepay

import "errors"

var (
	ErrInvalidWebhook     = errors.New("sepay: invalid webhook")
	ErrIgnoredWebhook     = errors.New("sepay: webhook ignored")
	ErrMissingCredentials = errors.New("sepay: missing credentials")
	ErrAccountNotLinked   = errors.New("sepay: bank account is not linked to the authenticated SePay company")
)
