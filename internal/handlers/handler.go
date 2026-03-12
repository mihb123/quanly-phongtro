package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mihb123/quanly-phongtro/config"
)

type Handler struct {
	db  *sql.DB
	cfg *config.Config
}

func New(db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
	}
}

func HealthCheck(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func Home(c *gin.Context) {
	c.File("./web/static/index.html")
}

type apiResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
	Error   any  `json:"error,omitempty"`
}

func writeJSON(c *gin.Context, status int, payload any) {
	res := apiResponse{
		Success: status < http.StatusBadRequest,
	}

	if res.Success {
		res.Data = payload
	} else {
		res.Error = payload
	}

	c.JSON(status, res)
}

func writeError(c *gin.Context, status int, message string) {
	writeJSON(c, status, message)
}
