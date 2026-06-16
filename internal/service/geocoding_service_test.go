package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeocodingService_GoogleMaps(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") == "bad_key" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Query().Get("key") == "invalid_json" {
			w.Write([]byte("{invalid json}"))
			return
		}
		if r.URL.Query().Get("key") == "api_error" {
			w.Write([]byte(`{"status": "REQUEST_DENIED", "results": []}`))
			return
		}
		// success
		w.Write([]byte(`{
			"status": "OK",
			"results": [{"formatted_address": "123 Main St, Hanoi"}]
		}`))
	}))
	defer ts.Close()

	originalBase := googleGeocodeBaseURL
	googleGeocodeBaseURL = ts.URL
	defer func() { googleGeocodeBaseURL = originalBase }()

	t.Run("success", func(t *testing.T) {
		svc := NewGeocodingService("good_key")
		addr, source, err := svc.ReverseGeocode(21.0, 105.8)
		assert.NoError(t, err)
		assert.Equal(t, "google", source)
		assert.Equal(t, "123 Main St, Hanoi", addr)
	})

	t.Run("http status not ok", func(t *testing.T) {
		svc := NewGeocodingService("bad_key").(*geocodingServiceImpl)
		_, err := svc.googleMapsReverseGeocode(21.0, 105.8)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "google maps api returned status")
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := NewGeocodingService("invalid_json").(*geocodingServiceImpl)
		_, err := svc.googleMapsReverseGeocode(21.0, 105.8)
		assert.Error(t, err)
	})

	t.Run("api status not ok", func(t *testing.T) {
		svc := NewGeocodingService("api_error").(*geocodingServiceImpl)
		_, err := svc.googleMapsReverseGeocode(21.0, 105.8)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "google maps api status")
	})

	t.Run("client get error", func(t *testing.T) {
		old := googleGeocodeBaseURL
		googleGeocodeBaseURL = "http://127.0.0.1:0"
		svc := NewGeocodingService("key").(*geocodingServiceImpl)
		svc.client.Transport = &errorTransport{}
		_, err := svc.googleMapsReverseGeocode(21.0, 105.8)
		assert.Error(t, err)
		googleGeocodeBaseURL = old
	})
}

func TestGeocodingService_OSM(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lat := r.URL.Query().Get("lat")
		if lat == "1.000000" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if lat == "2.000000" {
			w.Write([]byte("{invalid json}"))
			return
		}
		if lat == "3.000000" {
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		if lat == "4.000000" {
			w.Write([]byte(`{"display_name": ""}`))
			return
		}
		// success
		w.Write([]byte(`{"display_name": "456 Side St, HCM"}`))
	}))
	defer ts.Close()

	originalBase := osmGeocodeBaseURL
	osmGeocodeBaseURL = ts.URL
	defer func() { osmGeocodeBaseURL = originalBase }()

	svc := NewGeocodingService("") // empty key -> OSM

	t.Run("success", func(t *testing.T) {
		addr, source, err := svc.ReverseGeocode(0.0, 0.0)
		assert.NoError(t, err)
		assert.Equal(t, "osm", source)
		assert.Equal(t, "456 Side St, HCM", addr)
	})

	t.Run("http not ok", func(t *testing.T) {
		svcImpl := svc.(*geocodingServiceImpl)
		_, err := svcImpl.osmReverseGeocode(1.0, 0.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "osm api returned status")
	})

	t.Run("invalid json", func(t *testing.T) {
		svcImpl := svc.(*geocodingServiceImpl)
		_, err := svcImpl.osmReverseGeocode(2.0, 0.0)
		assert.Error(t, err)
	})

	t.Run("osm error field", func(t *testing.T) {
		svcImpl := svc.(*geocodingServiceImpl)
		_, err := svcImpl.osmReverseGeocode(3.0, 0.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "osm api error")
	})

	t.Run("empty display name", func(t *testing.T) {
		svcImpl := svc.(*geocodingServiceImpl)
		_, err := svcImpl.osmReverseGeocode(4.0, 0.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty display name")
	})

	t.Run("client do error", func(t *testing.T) {
		old := osmGeocodeBaseURL
		osmGeocodeBaseURL = "http://127.0.0.1:0"
		svcImpl := svc.(*geocodingServiceImpl)
		svcImpl.client.Transport = &errorTransport{}
		_, err := svcImpl.osmReverseGeocode(5.0, 0.0)
		assert.Error(t, err)
		osmGeocodeBaseURL = old
	})

	t.Run("new request error", func(t *testing.T) {
		old := osmGeocodeBaseURL
		osmGeocodeBaseURL = "::invalid::"
		svcImpl := svc.(*geocodingServiceImpl)
		_, err := svcImpl.osmReverseGeocode(6.0, 0.0)
		assert.Error(t, err)
		osmGeocodeBaseURL = old
	})
}

func TestGeocodingService_Fallback(t *testing.T) {
	googleFailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer googleFailServer.Close()

	osmSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"display_name": "Fallback City, VN"}`))
	}))
	defer osmSuccessServer.Close()

	originalGoogle := googleGeocodeBaseURL
	originalOSM := osmGeocodeBaseURL
	googleGeocodeBaseURL = googleFailServer.URL
	osmGeocodeBaseURL = osmSuccessServer.URL
	defer func() {
		googleGeocodeBaseURL = originalGoogle
		osmGeocodeBaseURL = originalOSM
	}()

	t.Run("fallback to OSM when Google fails", func(t *testing.T) {
		svc := NewGeocodingService("fake_google_key")
		addr, source, err := svc.ReverseGeocode(10.0, 10.0)
		assert.NoError(t, err)
		assert.Equal(t, "osm", source)
		assert.Equal(t, "Fallback City, VN", addr)
	})

}

type errorTransport struct{}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, assert.AnError
}
