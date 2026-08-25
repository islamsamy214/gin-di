package middlewares

import (
	"log"
	"net/http"
	"strings"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

// BearerPrefix is the authorization scheme this middleware accepts.
const BearerPrefix = "Bearer "

// Authenticate returns a handler bound to the given auth service, so the
// middleware depends on an injected collaborator rather than package state.
func Authenticate(auth *services.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// CutPrefix validates and strips in one step; slicing by prefix length
		// panics on any header shorter than the prefix.
		rawToken, ok := strings.CutPrefix(ctx.Request.Header.Get("Authorization"), BearerPrefix)

		if !ok || rawToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			return
		}

		// Parse and validate the token
		claims, err := auth.ParseToken(rawToken)
		if err != nil {
			// Log the cause, but never leak parser internals to the caller.
			log.Printf("auth: rejecting token: %v", err)

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			return
		}

		ctx.Set("userId", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Next()
	}
}
