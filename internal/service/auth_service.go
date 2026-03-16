package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenProvider interface {
	Generate(userID, email string) (string, error)
}

type AuthServiceImpl struct {
	users  model.UserRepository
	hasher PasswordHasher
	tokens TokenProvider
}

type AuthService interface {
	Register(ctx context.Context, in RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, in LoginInput) (*AuthOutput, error)
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	FullName    string `json:"full_name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	IsActivated bool   `json:"is_activated"`
	Token       string `json:"token"`
}

func NewAuthService(users model.UserRepository, hasher PasswordHasher, tokens TokenProvider) *AuthServiceImpl {
	return &AuthServiceImpl{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)

	if !isValidEmail(email) || len(password) < 6 {
		return nil, ErrInvalidInput
	}

	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}

	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         model.RoleManager,
		FullName:     fullName,
		Phone:        phone,
		IsActivated:  false,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	token, err := s.tokens.Generate(newUser.ID, newUser.Email)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:      newUser.ID,
		Email:       newUser.Email,
		Role:        string(newUser.Role),
		FullName:    newUser.FullName,
		Phone:       newUser.Phone,
		IsActivated: newUser.IsActivated,
		Token:       token,
	}, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)

	if !isValidEmail(email) || password == "" {
		return nil, ErrInvalidInput
	}

	existingUser, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if err := s.hasher.Compare(existingUser.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(existingUser.ID, existingUser.Email)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:      existingUser.ID,
		Email:       existingUser.Email,
		Role:        string(existingUser.Role),
		FullName:    existingUser.FullName,
		Phone:       existingUser.Phone,
		IsActivated: existingUser.IsActivated,
		Token:       token,
	}, nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
