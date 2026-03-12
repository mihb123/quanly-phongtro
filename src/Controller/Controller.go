package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type apiResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
	Error   any  `json:"error,omitempty"`
}

func writeJSON(ginContext *gin.Context, status int, payload any) {
	res := apiResponse{
		Success: status < http.StatusBadRequest,
	}

	if res.Success {
		res.Data = payload
	} else {
		res.Error = payload
	}

	ginContext.JSON(status, res)
}

func writeError(ginContext *gin.Context, status int, message string) {
	writeJSON(ginContext, status, message)
}

func HealthCheck(ginContext *gin.Context) {
	writeJSON(ginContext, http.StatusOK, gin.H{"status": "ok"})
}

func Home(ginContext *gin.Context) {
	ginContext.File("./web/static/index.html")
}
