package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/internal/db"
	"github.com/mihb123/quanly-phongtro/internal/repository/auth"
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

	repo := auth.NewAuthSessionRepository(database)
	ctx := context.Background()

	session, err := repo.FindByToken(ctx, "non-existent", "94c0e326-d2af-4ff2-8c1e-578cfb62114b")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: %v\n", session)
	}
}
