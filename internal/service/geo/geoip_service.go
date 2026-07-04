package geo

import (
	"fmt"
	"net"

	"github.com/mihb123/quanly-phongtro/internal/service/logger"
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
// thì degrade về "Unknown" giống NewGeoIPService.
func NewGeoIPServiceFromBytes(data []byte) GeoIPService {
	if len(data) == 0 {
		logger.Warn(nil, 0, "Embedded GeoIP database is empty. Location tracking will gracefully degrade to 'Unknown'.", nil)
		return &GeoIPServiceImpl{db: nil}
	}

	db, err := geoip2.FromBytes(data)
	if err != nil {
		logger.Error(nil, 0, "Failed to open embedded GeoIP database. Location tracking will gracefully degrade.", err)
		return &GeoIPServiceImpl{db: nil}
	}

	return &GeoIPServiceImpl{db: db}
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
