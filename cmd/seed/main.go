package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/internal/db"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/auth"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	userRepository := auth.NewUserRepository(database)
	hasher := security.NewBcryptHasher()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testUsers := []struct {
		email    string
		password string
		role     model.Role
		fullName string
		phone    string
	}{
		{
			email:    "manager@test.com",
			password: "password12300",
			role:     model.RoleManager,
			fullName: "Test Manager",
			phone:    "0123456789",
		},
		{
			email:    "tenant@test.com",
			password: "password12300",
			role:     model.RoleTenant,
			fullName: "Test Tenant",
			phone:    "0987654321",
		},
	}

	for _, tu := range testUsers {
		existingUser, err := userRepository.GetByEmail(ctx, tu.email)
		if err == nil && existingUser != nil {
			fmt.Printf("User %s already exists, skipping...\n", tu.email)
			continue
		}

		passwordHash, err := hasher.Hash(tu.password)
		if err != nil {
			log.Fatalf("failed to hash password for %s: %v", tu.email, err)
		}

		user := &model.User{
			Email:        tu.email,
			PasswordHash: passwordHash,
			Role:         tu.role,
			FullName:     tu.fullName,
			Phone:        tu.phone,
			IsActivated:  true,
		}

		if err := userRepository.Create(ctx, user); err != nil {
			log.Fatalf("failed to create user %s: %v", tu.email, err)
		}

		fmt.Printf("Successfully created user: %s (Role: %s)\n", tu.email, tu.role)
	}
}
