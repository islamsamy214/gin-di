package middlewares

import (
	"fmt"
	"strings"
	"web-app/app/auth"
	"web-app/app/exceptions"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

// BearerPrefix is the authorization scheme this middleware accepts.
const BearerPrefix = "Bearer "

/*
 * Authenticate returns a handler bound to the given auth service, so the
 * middleware depends on an injected collaborator rather than package state.
 *
 * Every rejection is reported through ctx.Error rather than written here, so the
 * exception handler renders it in the same envelope as every other failure and
 * records the cause in one place.
 *
 * @param authService The service the presented token is verified against.
 */
func Authenticate(authService *services.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// CutPrefix validates and strips in one step; slicing by prefix length
		// panics on any header shorter than the prefix.
		rawToken, ok := strings.CutPrefix(ctx.Request.Header.Get("Authorization"), BearerPrefix)

		if !ok || rawToken == "" {
			reject(ctx, fmt.Errorf("authorization header missing or not a %sscheme", BearerPrefix))

			return
		}

		claims, err := authService.ParseToken(rawToken)
		if err != nil {
			// The cause reaches the log through the exception's wrapped error and
			// never reaches the caller: parser internals distinguish an expired
			// token from a forged one, which is not something to volunteer.
			reject(ctx, fmt.Errorf("rejecting token: %w", err))

			return
		}

		auth.SetUser(ctx, claims.UserID, claims.Username)

		ctx.Next()
	}
}

/*
 * reject aborts the request as unauthorized, carrying the cause for the log.
 *
 * @param ctx   The request to reject.
 * @param cause Why it was rejected. Logged, never sent.
 */
func reject(ctx *gin.Context, cause error) {
	unauthorized := exceptions.NewUnauthorized()
	unauthorized.Err = cause

	_ = ctx.Error(unauthorized)
	ctx.Abort()
}
