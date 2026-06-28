package service_test

import (
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestNewGeoIPServiceFromBytes(t *testing.T) {
	// Dữ liệu rỗng → degrade về "Unknown"
	svc := service.NewGeoIPServiceFromBytes(nil)
	assert.NotNil(t, svc)
	assert.Equal(t, "Unknown", svc.LookupLocation("8.8.8.8"))
	svc.Close()

	// Dữ liệu mmdb không hợp lệ → degrade về "Unknown"
	svc2 := service.NewGeoIPServiceFromBytes([]byte("invalid data"))
	assert.NotNil(t, svc2)
	assert.Equal(t, "Unknown", svc2.LookupLocation("8.8.8.8"))
	svc2.Close()
}

func TestGeoIPService_LookupLocation(t *testing.T) {
	// Tạo service degraded (không có DB)
	svc := service.NewGeoIPServiceFromBytes(nil)

	t.Run("empty IP", func(t *testing.T) {
		assert.Equal(t, "Unknown", svc.LookupLocation(""))
	})

	t.Run("invalid IP format", func(t *testing.T) {
		assert.Equal(t, "Unknown", svc.LookupLocation("not-an-ip"))
	})

	t.Run("loopback IP", func(t *testing.T) {
		assert.Equal(t, "Local Network", svc.LookupLocation("127.0.0.1"))
	})

	t.Run("private IP", func(t *testing.T) {
		assert.Equal(t, "Local Network", svc.LookupLocation("192.168.1.100"))
		assert.Equal(t, "Local Network", svc.LookupLocation("10.0.0.5"))
	})

	t.Run("public IP on degraded service", func(t *testing.T) {
		assert.Equal(t, "Unknown", svc.LookupLocation("8.8.8.8"))
	})
}
