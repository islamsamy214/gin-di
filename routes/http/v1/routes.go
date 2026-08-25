package v1

import (
	"web-app/app/http/middlewares"
	"web-app/app/services"
	"web-app/routes/http/v1/auth"
	"web-app/routes/http/v1/events"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts every v1 domain onto the given group.
 *
 * Takes a *gin.RouterGroup rather than the engine so the caller owns the prefix
 * and this table stays mountable anywhere. Adding a domain is one Register call
 * here plus a new package; no existing domain is touched.
 *
 * @param router      The group this version hangs from, already prefixed.
 * @param authService The service injected down into each domain.
 */
func Register(router *gin.RouterGroup, authService *services.AuthService) {
	// Public surface.
	auth.Register(router, authService)

	// Authenticated surface: the middleware hangs on the group, so every domain
	// mounted below inherits it and none can be exposed unprotected by accident.
	protected := router.Group("")
	protected.Use(middlewares.Authenticate(authService))

	events.Register(protected)
}
