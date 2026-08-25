package events

import (
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
 */
func Register(router *gin.RouterGroup) {
	controller := controllers.NewController()

	events := router.Group("/events")
	events.GET("", controller.Index)
	events.POST("", controller.Create)
}
