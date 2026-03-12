package middleware

import "github.com/gin-gonic/gin"

func CORS() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ginContext.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ginContext.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ginContext.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
		ginContext.Writer.Header().Set("Access-Control-Expose-Headers", "X-Page, X-Page-Size, X-Total-Count, X-Total-Pages")

		if ginContext.Request.Method == "OPTIONS" {
			ginContext.AbortWithStatus(204)
			return
		}

		ginContext.Next()
	}
}

func JSONContentType() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ginContext.Header("Content-Type", "application/json")
		ginContext.Next()
	}
}
