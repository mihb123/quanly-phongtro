package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mihb123/quanly-phongtro/src/Request"
	"github.com/mihb123/quanly-phongtro/src/Service"
)

type AuthController struct {
	authService service.AuthService
}

func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (controller *AuthController) Register(ginContext *gin.Context) {
	var registerRequest request.RegisterRequest
	if err := ginContext.ShouldBindJSON(&registerRequest); err != nil {
		writeError(ginContext, http.StatusBadRequest, err.Error())
		return
	}

	requestContext, cancelTimeout := context.WithTimeout(ginContext.Request.Context(), 5*time.Second)
	defer cancelTimeout()

	authResult, err := controller.authService.Register(requestContext, registerRequest.Email, registerRequest.FullName, registerRequest.Password)
	if err != nil {
		if service.IsDuplicateEmailError(err) {
			writeError(ginContext, http.StatusConflict, "email already exists")
			return
		}
		writeError(ginContext, http.StatusInternalServerError, "failed to create account")
		return
	}

	writeJSON(ginContext, http.StatusCreated, gin.H{
		"message": "register success",
		"email":   authResult.Email,
	})
}

func (controller *AuthController) Login(ginContext *gin.Context) {
	var loginRequest request.LoginRequest
	if err := ginContext.ShouldBindJSON(&loginRequest); err != nil {
		writeError(ginContext, http.StatusBadRequest, err.Error())
		return
	}

	requestContext, cancelTimeout := context.WithTimeout(ginContext.Request.Context(), 5*time.Second)
	defer cancelTimeout()

	authResult, err := controller.authService.Login(requestContext, loginRequest.Email, loginRequest.Password)
	if err != nil {
		if err == service.ErrorInvalidCredentials {
			writeError(ginContext, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(ginContext, http.StatusInternalServerError, "failed to login")
		return
	}

	writeJSON(ginContext, http.StatusOK, gin.H{
		"message": "login success",
		"token":   authResult.Token,
		"user": gin.H{
			"id":        authResult.User.ID,
			"email":     authResult.User.Email,
			"full_name": authResult.User.FullName,
			"role":      authResult.User.Role,
		},
	})
}
