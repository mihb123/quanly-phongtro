package service

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

	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/src/Model"
	"github.com/mihb123/quanly-phongtro/src/Repository"
	"golang.org/x/crypto/bcrypt"
)

type RegisterResult struct {
	Email string
}

type LoginResult struct {
	Token string
	User  *model.User
}

type AuthService interface {
	Register(requestContext context.Context, email, fullName, password string) (*RegisterResult, error)
	Login(requestContext context.Context, email, password string) (*LoginResult, error)
}

type authService struct {
	userRepository repository.UserRepository
}

func NewAuthService(userRepository repository.UserRepository) AuthService {
	return &authService{userRepository: userRepository}
}

func (service *authService) Register(requestContext context.Context, email, fullName, password string) (*RegisterResult, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if err := service.userRepository.Create(requestContext, email, fullName, string(passwordHash)); err != nil {
		return nil, err
	}

	return &RegisterResult{Email: email}, nil
}

func (service *authService) Login(requestContext context.Context, email, password string) (*LoginResult, error) {
	foundUser, err := service.userRepository.FindByEmail(requestContext, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrorInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(password)); err != nil {
		return nil, ErrorInvalidCredentials
	}

	currentTimeUnix := time.Now().Unix()
	tokenClaims := map[string]any{
		"sub":   foundUser.ID,
		"email": foundUser.Email,
		"role":  foundUser.Role,
		"iat":   currentTimeUnix,
		"exp":   currentTimeUnix + 86400,
	}

	token, err := signJsonWebToken(tokenClaims, []byte(getJsonWebTokenSecret()))
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, User: foundUser}, nil
}

func IsDuplicateEmailError(err error) bool {
	postgresError, ok := err.(*pq.Error)
	return ok && postgresError.Code == "23505"
}

func signJsonWebToken(tokenClaims map[string]any, secret []byte) (string, error) {
	headerJsonBytes, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payloadJsonBytes, err := json.Marshal(tokenClaims)
	if err != nil {
		return "", err
	}

	headerSegment := base64.RawURLEncoding.EncodeToString(headerJsonBytes)
	payloadSegment := base64.RawURLEncoding.EncodeToString(payloadJsonBytes)
	unsignedToken := headerSegment + "." + payloadSegment

	hmacHandler := hmac.New(sha256.New, secret)
	_, _ = hmacHandler.Write([]byte(unsignedToken))
	signatureSegment := base64.RawURLEncoding.EncodeToString(hmacHandler.Sum(nil))

	return unsignedToken + "." + signatureSegment, nil
}

func getJsonWebTokenSecret() string {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret != "" {
		return secret
	}
	return "dev-secret"
}
