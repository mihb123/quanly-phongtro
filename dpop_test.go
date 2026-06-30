package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func main() {
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "localhost:8080"
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", "localhost:5173")

	trustedProxies := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	htu := security.BuildDPoPHTU(req, trustedProxies)
	fmt.Println("Constructed HTU:", htu)
}
