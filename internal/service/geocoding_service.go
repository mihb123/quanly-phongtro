package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GeocodingService interface {
	ReverseGeocode(lat, lng float64) (string, error)
}

type geocodingServiceImpl struct {
	apiKey string
	client *http.Client
}

func NewGeocodingService(apiKey string) GeocodingService {
	return &geocodingServiceImpl{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *geocodingServiceImpl) ReverseGeocode(lat, lng float64) (string, error) {
	if s.apiKey != "" {
		return s.googleMapsReverseGeocode(lat, lng)
	}
	return s.osmReverseGeocode(lat, lng)
}

var (
	googleGeocodeBaseURL = "https://maps.googleapis.com/maps/api/geocode/json"
	osmGeocodeBaseURL    = "https://nominatim.openstreetmap.org/reverse"
)

type googleGeocodeResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
	} `json:"results"`
	Status string `json:"status"`
}

func (s *geocodingServiceImpl) googleMapsReverseGeocode(lat, lng float64) (string, error) {
	url := fmt.Sprintf("%s?latlng=%f,%f&key=%s&language=vi", googleGeocodeBaseURL, lat, lng, s.apiKey)
	resp, err := s.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google maps api returned status: %d", resp.StatusCode)
	}

	var data googleGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Status != "OK" || len(data.Results) == 0 {
		return "", fmt.Errorf("google maps api status: %s", data.Status)
	}

	return data.Results[0].FormattedAddress, nil
}

type osmGeocodeResponse struct {
	DisplayName string `json:"display_name"`
	Error       string `json:"error"`
}

func (s *geocodingServiceImpl) osmReverseGeocode(lat, lng float64) (string, error) {
	url := fmt.Sprintf("%s?format=json&lat=%f&lon=%f&accept-language=vi", osmGeocodeBaseURL, lat, lng)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// OSM Requires a descriptive User-Agent
	req.Header.Set("User-Agent", "quanly-phongtro/1.0 (https://github.com/mihb123/quanly-phongtro)")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("osm api returned status: %d", resp.StatusCode)
	}

	var data osmGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Error != "" {
		return "", fmt.Errorf("osm api error: %s", data.Error)
	}

	if data.DisplayName == "" {
		return "", fmt.Errorf("osm api returned empty display name")
	}

	return data.DisplayName, nil
}
