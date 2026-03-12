package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/src/Middleware"
	"github.com/mihb123/quanly-phongtro/src/Routes"
)

type Server struct {
	httpServer    *http.Server
	configuration *config.Config
}

func New(configuration *config.Config, databaseConnection *sql.DB) *Server {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	routes.Register(router, configuration, databaseConnection)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", configuration.Host, configuration.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(configuration.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(configuration.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(configuration.IdleTimeoutSec) * time.Second,
	}

	return &Server{
		httpServer:    httpServer,
		configuration: configuration,
	}
}

func (server *Server) Start() error {
	return server.httpServer.ListenAndServe()
}

func (server *Server) Shutdown(shutdownContext context.Context) error {
	return server.httpServer.Shutdown(shutdownContext)
}
