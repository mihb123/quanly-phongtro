package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestNewGeoIPService(t *testing.T) {
	// Path does not exist
	svc := service.NewGeoIPService("invalid_path.mmdb")
	assert.NotNil(t, svc)
	assert.Equal(t, "Unknown", svc.LookupLocation("8.8.8.8"))
	svc.Close()

	// Path exists but invalid DB
	tmpFile := filepath.Join(t.TempDir(), "invalid.mmdb")
	os.WriteFile(tmpFile, []byte("invalid data"), 0644)

	svc2 := service.NewGeoIPService(tmpFile)
	assert.NotNil(t, svc2)
	assert.Equal(t, "Unknown", svc2.LookupLocation("8.8.8.8"))
	svc2.Close()
}

func TestGeoIPService_LookupLocation(t *testing.T) {
	// Create a degraded service
	svc := service.NewGeoIPService("invalid_path.mmdb")

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
