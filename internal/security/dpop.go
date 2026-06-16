package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidDPoPProof = errors.New("invalid dpop proof")
	ErrReplayDPoPProof  = errors.New("dpop proof jti is reused")
)

// ReplayCache is a simple in-memory cache to store used jti values
type ReplayCache struct {
	mu    sync.RWMutex
	cache map[string]time.Time
}

func NewReplayCache() *ReplayCache {
	rc := &ReplayCache{
		cache: make(map[string]time.Time),
	}
	// Background cleanup every minute
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			rc.cleanup()
		}
	}()
	return rc
}

func (rc *ReplayCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	now := time.Now()
	for k, v := range rc.cache {
		if now.After(v) {
			delete(rc.cache, k)
		}
	}
}

func (rc *ReplayCache) CheckAndStore(jti string, expiresAt time.Time) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if _, exists := rc.cache[jti]; exists {
		return false // Replay detected
	}
	rc.cache[jti] = expiresAt
	return true
}

var globalReplayCache = NewReplayCache()

type DPoPClaims struct {
	Htu string `json:"htu"`
	Htm string `json:"htm"`
	Jti string `json:"jti"`
	Ath string `json:"ath,omitempty"`
	jwt.RegisteredClaims
}

// ExtractJWKThumbprint extracts the JWK thumbprint from a DPoP proof string without verifying signature
// This is used during login/register when we don't have an access token yet but need the public key
func ExtractJWKThumbprint(dpopStr string) (string, error) {
	if dpopStr == "" {
		return "", nil
	}

	token, _, err := new(jwt.Parser).ParseUnverified(dpopStr, &DPoPClaims{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidDPoPProof, err)
	}

	jwk, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%w: missing jwk header", ErrInvalidDPoPProof)
	}

	return calculateThumbprint(jwk)
}

// VerifyDPoPProof parses and verifies a DPoP proof
func VerifyDPoPProof(dpopStr, htm, htu, accessToken string) (string, error) {
	if dpopStr == "" {
		return "", fmt.Errorf("%w: missing proof", ErrInvalidDPoPProof)
	}

	// 1. Parse and verify signature
	var claims DPoPClaims
	token, err := jwt.ParseWithClaims(dpopStr, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Header["typ"] != "dpop+jwt" {
			return nil, fmt.Errorf("invalid typ header")
		}

		alg := token.Method.Alg()
		if alg != "ES256" && alg != "RS256" {
			return nil, fmt.Errorf("unsupported alg: %s", alg)
		}

		jwk, ok := token.Header["jwk"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing jwk header")
		}

		return parseJWKToPublicKey(jwk)
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("%w: %v", ErrInvalidDPoPProof, err)
	}

	// 2. Validate Claims (htm, htu)
	if claims.Htm != htm {
		return "", fmt.Errorf("%w: htm mismatch", ErrInvalidDPoPProof)
	}

	// Normalize URL by removing query params if needed, but standard DPoP requires exact match of scheme://host:port/path
	if claims.Htu != htu {
		return "", fmt.Errorf("%w: htu mismatch", ErrInvalidDPoPProof)
	}

	// 3. Validate iat (Time window: +/- 5 minutes)
	if claims.IssuedAt == nil {
		return "", fmt.Errorf("%w: missing iat", ErrInvalidDPoPProof)
	}
	iat := claims.IssuedAt.Time
	if time.Since(iat).Abs() > 5*time.Minute {
		return "", fmt.Errorf("%w: proof expired or too far in future", ErrInvalidDPoPProof)
	}

	// 4. Validate jti (Replay Cache)
	if claims.Jti == "" {
		return "", fmt.Errorf("%w: missing jti", ErrInvalidDPoPProof)
	}
	if !globalReplayCache.CheckAndStore(claims.Jti, time.Now().Add(5*time.Minute)) {
		return "", ErrReplayDPoPProof
	}

	// 5. Validate ath (Access Token Hash) if accessToken is provided
	if accessToken != "" {
		if claims.Ath == "" {
			return "", fmt.Errorf("%w: missing ath", ErrInvalidDPoPProof)
		}
		hash := sha256.Sum256([]byte(accessToken))
		expectedAth := base64.RawURLEncoding.EncodeToString(hash[:])
		if claims.Ath != expectedAth {
			return "", fmt.Errorf("%w: ath mismatch", ErrInvalidDPoPProof)
		}
	}

	// 6. Calculate and return JWK Thumbprint (jkt)
	jwk, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%w: missing jwk header", ErrInvalidDPoPProof)
	}
	return calculateThumbprint(jwk)
}

func parseJWKToPublicKey(jwk map[string]interface{}) (interface{}, error) {
	kty, _ := jwk["kty"].(string)

	if kty == "EC" {
		crv, _ := jwk["crv"].(string)
		if crv != "P-256" {
			return nil, fmt.Errorf("unsupported curve: %s", crv)
		}

		xBase64, _ := jwk["x"].(string)
		yBase64, _ := jwk["y"].(string)

		xBytes, err := base64.RawURLEncoding.DecodeString(xBase64)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(yBase64)
		if err != nil {
			return nil, err
		}

		curve := elliptic.P256()
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		if !curve.IsOnCurve(x, y) {
			return nil, fmt.Errorf("invalid EC public key point")
		}

		pubKey := &ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}
		return pubKey, nil
	}

	return nil, fmt.Errorf("unsupported key type: %s", kty)
}

func calculateThumbprint(jwk map[string]interface{}) (string, error) {
	// RFC 7638 JSON Web Key (JWK) Thumbprint
	// Required fields for EC public keys in lexicographic order: crv, kty, x, y

	kty, _ := jwk["kty"].(string)
	if kty != "EC" {
		return "", fmt.Errorf("only EC keys supported for thumbprint")
	}

	crv, _ := jwk["crv"].(string)
	x, _ := jwk["x"].(string)
	y, _ := jwk["y"].(string)

	// Lexicographic order and no spaces
	jsonString := fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`, crv, kty, x, y)

	hash := sha256.Sum256([]byte(jsonString))
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}
