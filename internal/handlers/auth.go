package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Register(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, 400, err.Error())
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(c, 500, "failed to hash password")
		return
	}

	const query = `
		INSERT INTO users (email, full_name, password_hash, role)
		VALUES ($1, $2, $3, 'MANAGER')
	`

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.db.ExecContext(ctx, query, req.Email, req.FullName, string(passwordHash)); err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			writeError(c, 409, "email already exists")
			return
		}

		writeError(c, 500, "failed to create account")
		return
	}

	writeJSON(c, 201, gin.H{
		"message": "register success",
		"email":   req.Email,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, 400, err.Error())
		return
	}

	const query = `
		SELECT id::text, email, COALESCE(full_name, ''), role, password_hash
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var userID, email, fullName, role, passwordHash string
	if err := h.db.QueryRowContext(ctx, query, req.Email).Scan(&userID, &email, &fullName, &role, &passwordHash); err != nil {
		if err == sql.ErrNoRows {
			writeError(c, 401, "invalid credentials")
			return
		}
		writeError(c, 500, "failed to login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(c, 401, "invalid credentials")
		return
	}

	now := time.Now().Unix()
	claims := map[string]any{
		"sub":   userID,
		"email": email,
		"role":  role,
		"iat":   now,
		"exp":   now + 86400,
	}

	token, err := signJWT(claims, []byte(jwtSecret()))
	if err != nil {
		writeError(c, 500, "failed to create token")
		return
	}

	writeJSON(c, 200, gin.H{
		"message": "login success",
		"token":   token,
		"user": gin.H{
			"id":        userID,
			"email":     email,
			"full_name": fullName,
			"role":      role,
		},
	})
}

func signJWT(claims map[string]any, secret []byte) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + signature, nil
}

func jwtSecret() string {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret != "" {
		return secret
	}
	return "dev-secret"
}
