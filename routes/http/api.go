package http

import (
	"web-app/app/http/controllers"
	"web-app/app/http/middlewares"

	"github.com/gin-gonic/gin"
)

func Regester(route *gin.Engine) {
	// authentication routes
	authController := controllers.NewAuthController()
	route.POST("/login", authController.Login)

	// events routes
	eventController := controllers.NewEventController()
	route.GET("/events", middlewares.Authenticate, eventController.Index)
	route.POST("/events", middlewares.Authenticate, eventController.Create)

	// // group it to middleware
	// auth := route.Group("/events")
	// auth.Use(middlewares.Authenticate)
	// auth.POST("", eventController.Create)
	// auth.PUT("/:id", eventController.Update)
	// auth.DELETE("/:id", eventController.Delete)

}
