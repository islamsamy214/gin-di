package v1

import (
	controllers "web-app/app/http/controllers/v1"
	"web-app/app/http/middlewares"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts the v1 API onto the given group.
 *
 * Takes a *gin.RouterGroup rather than the engine so the caller owns the prefix
 * and this table stays mountable anywhere. Adding a sibling version means a new
 * package like this one, not an edit here.
 *
 * @param router The group this version hangs from, already prefixed.
 * @param auth   The service its controllers and middleware are built with.
 */
func Register(router *gin.RouterGroup, auth *services.AuthService) {
	// authentication routes
	authController := controllers.NewAuthController(auth)
	router.POST("/login", authController.Login)

	// Authenticated surface: the middleware hangs on the group, so every route
	// added below inherits it and none can be registered unprotected by accident.
	protected := router.Group("")
	protected.Use(middlewares.Authenticate(auth))

	// events routes
	eventController := controllers.NewEventController()
	events := protected.Group("/events")
	events.GET("", eventController.Index)
	events.POST("", eventController.Create)
}
