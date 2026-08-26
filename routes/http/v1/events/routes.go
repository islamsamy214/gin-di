package events

import (
	"web-app/app/container"
	controllers "web-app/app/http/controllers/v1/events"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts the events resource.
 *
 * The caller owns authentication: this expects an already-protected group, so
 * the domain never has to remember to apply the middleware itself.
 *
 * @param router The protected group these routes hang from.
 * @param c      The resolved application the controller is built from.
 */
func Register(router *gin.RouterGroup, c *container.Container) {
	controller := controllers.NewController(c.DB())

	events := router.Group("/events")
	events.GET("", controller.Index)
	events.POST("", controller.Create)
}
