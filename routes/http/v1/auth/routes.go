package auth

import (
	"web-app/app/container"
	controllers "web-app/app/http/controllers/v1/auth"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts the public authentication routes.
 *
 * @param router The group these routes hang from.
 * @param c      The resolved application the controller is built from.
 */
func Register(router *gin.RouterGroup, c *container.Container) {
	controller := controllers.NewController(c.Auth(), c.Users())

	router.POST("/login", controller.Login)
}
