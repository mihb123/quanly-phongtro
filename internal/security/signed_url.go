package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	SignedURLExpiresParam   = "expires"
	SignedURLSignatureParam = "sig"
)

var ErrInvalidSignedURL = errors.New("invalid signed url")

// SignPath signs a URL path and expiration timestamp with HMAC-SHA256.
func SignPath(path string, expiresAt time.Time, key string) (url.Values, error) {
	if key == "" {
		return nil, errors.New("signed url key is required")
	}

	expires := strconv.FormatInt(expiresAt.Unix(), 10)
	signature, err := signPathPayload(path, expires, key)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set(SignedURLExpiresParam, expires)
	values.Set(SignedURLSignatureParam, signature)
	return values, nil
}

// VerifySignedPath validates a path signature and expiration timestamp.
func VerifySignedPath(path, expires, signature, key string, now time.Time) error {
	if key == "" || expires == "" || signature == "" {
		return ErrInvalidSignedURL
	}

	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid expires", ErrInvalidSignedURL)
	}
	if now.After(time.Unix(expiresUnix, 0)) {
		return fmt.Errorf("%w: expired", ErrInvalidSignedURL)
	}

	expected, err := signPathPayload(path, expires, key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("%w: signature mismatch", ErrInvalidSignedURL)
	}

	return nil
}

// signPathPayload returns the base64url HMAC for a path/expires pair.
func signPathPayload(path, expires, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	if _, err := mac.Write([]byte(path)); err != nil {
		return "", err
	}
	if _, err := mac.Write([]byte("\n")); err != nil {
		return "", err
	}
	if _, err := mac.Write([]byte(expires)); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
