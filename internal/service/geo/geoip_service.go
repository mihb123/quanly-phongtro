package geo

import (
	"fmt"
	"net"

	"github.com/mihb123/quanly-phongtro/pkg/apperr"
	"github.com/oschwald/geoip2-golang"
)

type GeoIPService interface {
	LookupLocation(ipAddress string) string
	Close()
}

type GeoIPServiceImpl struct {
	db *geoip2.Reader
}

// NewGeoIPServiceFromBytes khởi tạo service từ dữ liệu mmdb đã nhúng trong binary,
// giúp binary tự chứa GeoIP DB mà không cần file ngoài. Nếu dữ liệu rỗng hoặc lỗi
// thì degrade về "Unknown" giống NewGeoIPService: service trả về luôn dùng được,
// error (nếu có) chỉ để caller log cảnh báo.
func NewGeoIPServiceFromBytes(data []byte) (GeoIPService, error) {
	if len(data) == 0 {
		return &GeoIPServiceImpl{db: nil}, apperr.New(apperr.LayerService, "NewGeoIPServiceFromBytes", "embedded GeoIP database is empty; location degrades to Unknown")
	}

	db, err := geoip2.FromBytes(data)
	if err != nil {
		return &GeoIPServiceImpl{db: nil}, apperr.Service("NewGeoIPServiceFromBytes", fmt.Errorf("open embedded GeoIP database (location degrades to Unknown): %w", err))
	}

	return &GeoIPServiceImpl{db: db}, nil
}

func (s *GeoIPServiceImpl) LookupLocation(ipAddress string) string {
	if ipAddress == "" {
		return "Unknown"
	}

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return "Unknown"
	}

	// Check for local loopback or private IPs to return "Local Network"
	if ip.IsLoopback() || ip.IsPrivate() {
		return "Local Network"
	}

	if s.db == nil {
		return "Unknown"
	}

	record, err := s.db.City(ip)
	if err != nil || record == nil {
		return "Unknown"
	}

	city := record.City.Names["en"]
	country := record.Country.Names["en"]

	if city == "" && country == "" {
		return "Unknown"
	}

	if city == "" {
		return country
	}

	if country == "" {
		return city
	}

	return fmt.Sprintf("%s, %s", city, country)
}

func (s *GeoIPServiceImpl) Close() {
	if s.db != nil {
		s.db.Close()
	}
}
