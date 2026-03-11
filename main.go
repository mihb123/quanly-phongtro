package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/internal/database"
	"github.com/mihb123/quanly-phongtro/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer database.Disconnect(context.Background(), db)

	srv := server.New(cfg, db)

	go func() {
		log.Printf("server starting on http://%s:%s", cfg.Host, cfg.Port)
		if err := srv.Start(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
