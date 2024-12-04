package middlewares

import (
	"net/http"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

func Authenticate(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")

	if token == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "not authorized",
		})
		return
	}

	// Extract the token from the Authorization header
	token = token[len("Bearer "):]

	// Parse and validate the token
	claims, err := services.ParseToken(token)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		return
	}

	ctx.Set("userId", claims.UserID)
	ctx.Next()
}
