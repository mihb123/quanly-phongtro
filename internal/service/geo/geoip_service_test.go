package geo_test

import (
	"os"
	"testing"

	geosvc "github.com/mihb123/quanly-phongtro/internal/service/geo"

	"github.com/stretchr/testify/assert"
)

// embeddedGeoIPPath trỏ tới GeoIP DB thật đã nhúng, dùng để test các nhánh
// tra cứu thành công của LookupLocation (đường dẫn tương đối từ thư mục package).
const embeddedGeoIPPath = "../../assets/geoip/GeoLite2-City.mmdb"

// newRealGeoIPService nạp DB GeoIP thật từ file nhúng; nếu không tìm thấy thì
// skip test thay vì fail, tránh phụ thuộc cứng vào file lớn khi build ở nơi khác.
func newRealGeoIPService(t *testing.T) geosvc.GeoIPService {
	t.Helper()
	data, err := os.ReadFile(embeddedGeoIPPath)
	if err != nil {
		t.Skipf("GeoIP database not available at %s: %v", embeddedGeoIPPath, err)
	}
	return geosvc.NewGeoIPServiceFromBytes(data)
}

func TestNewGeoIPServiceFromBytes(t *testing.T) {
	// Dữ liệu rỗng → degrade về "Unknown"
	svc := geosvc.NewGeoIPServiceFromBytes(nil)
	assert.NotNil(t, svc)
	assert.Equal(t, "Unknown", svc.LookupLocation("8.8.8.8"))
	svc.Close()

	// Dữ liệu mmdb không hợp lệ → degrade về "Unknown"
	svc2 := geosvc.NewGeoIPServiceFromBytes([]byte("invalid data"))
	assert.NotNil(t, svc2)
	assert.Equal(t, "Unknown", svc2.LookupLocation("8.8.8.8"))
	svc2.Close()
}

func TestGeoIPService_LookupLocation(t *testing.T) {
	// Tạo service degraded (không có DB)
	svc := geosvc.NewGeoIPServiceFromBytes(nil)

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

// TestGeoIPService_LookupLocation_RealDB tra cứu trên DB GeoIP thật để phủ các
// nhánh thành công: có cả city+country, chỉ có country, và IP không có trong DB.
func TestGeoIPService_LookupLocation_RealDB(t *testing.T) {
	svc := newRealGeoIPService(t)
	defer svc.Close()

	t.Run("IP với cả city và country", func(t *testing.T) {
		assert.Equal(t, "East Finchley, United Kingdom", svc.LookupLocation("81.2.69.142"))
	})

	t.Run("IP chỉ có country", func(t *testing.T) {
		assert.Equal(t, "United States", svc.LookupLocation("8.8.8.8"))
	})

	t.Run("IP public không có trong DB → Unknown", func(t *testing.T) {
		assert.Equal(t, "Unknown", svc.LookupLocation("1.1.1.1"))
	})
}

// TestGeoIPService_LookupAfterClose kiểm tra sau khi Close(), việc tra cứu trên
// reader đã đóng trả về lỗi và service degrade an toàn về "Unknown".
func TestGeoIPService_LookupAfterClose(t *testing.T) {
	svc := newRealGeoIPService(t)
	svc.Close()
	assert.Equal(t, "Unknown", svc.LookupLocation("8.8.8.8"))
}

// TestGeoIPService_Close_Idempotent xác nhận Close() trên service degraded
// (db == nil) không panic, phủ nhánh db == nil của Close().
func TestGeoIPService_Close_Idempotent(t *testing.T) {
	svc := geosvc.NewGeoIPServiceFromBytes(nil)
	assert.NotPanics(t, func() { svc.Close() })
}
