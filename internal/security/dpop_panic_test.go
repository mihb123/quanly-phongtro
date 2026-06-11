package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"github.com/golang-jwt/jwt/v5"
)

func TestDPoPWithoutIat(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	
	claims := DPoPClaims{
		Htu: "/test",
		Htm: "GET",
		Jti: "random-jti",
		// NO IssuedAt
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	
	x := key.PublicKey.X.Bytes()
	y := key.PublicKey.Y.Bytes()
	
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(x),
		"y":   base64.RawURLEncoding.EncodeToString(y),
	}
	token.Header["jwk"] = jwk
	
	proof, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	
	// This should not panic
	_, err = VerifyDPoPProof(proof, "GET", "/test", "")
	t.Log(err)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
