package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/src/Database"
	"github.com/mihb123/quanly-phongtro/src/Server"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	databaseConnection, err := database.Connect(configuration.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer database.Disconnect(context.Background(), databaseConnection)

	httpServer := server.New(configuration, databaseConnection)

	go func() {
		log.Printf("server starting on http://%s:%s", configuration.Host, configuration.Port)
		if err := httpServer.Start(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	stopSignalChannel := make(chan os.Signal, 1)
	signal.Notify(stopSignalChannel, os.Interrupt, syscall.SIGTERM)
	<-stopSignalChannel

	shutdownContext, cancelShutdownTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdownTimeout()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
