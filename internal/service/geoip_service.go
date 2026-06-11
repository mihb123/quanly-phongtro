package service

import (
	"fmt"
	"net"
	"os"

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

func NewGeoIPService(dbPath string) GeoIPService {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		logger.Warn(nil, 0, fmt.Sprintf("GeoIP database not found at %s. Location tracking will gracefully degrade to 'Unknown'.", dbPath), nil)
		return &GeoIPServiceImpl{db: nil}
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		logger.Error(nil, 0, fmt.Sprintf("Failed to open GeoIP database at %s. Location tracking will gracefully degrade.", dbPath), err)
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
