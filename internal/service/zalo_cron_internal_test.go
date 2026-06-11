package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewZaloCronService_InvalidSchedule(t *testing.T) {
	old := TokenCheckCronSchedule
	TokenCheckCronSchedule = "invalid schedule"
	defer func() { TokenCheckCronSchedule = old }()

	// Should not panic, but log an error and return service
	svc := NewZaloCronService(nil, nil, []byte("123"))
	assert.NotNil(t, svc)
}
