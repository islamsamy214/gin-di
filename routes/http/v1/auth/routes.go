package auth

import (
	controllers "web-app/app/http/controllers/v1/auth"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts the public authentication routes.
 *
 * @param router      The group these routes hang from.
 * @param authService The service the controller is built with.
 */
func Register(router *gin.RouterGroup, authService *services.AuthService) {
	controller := controllers.NewController(authService)

	router.POST("/login", controller.Login)
}
