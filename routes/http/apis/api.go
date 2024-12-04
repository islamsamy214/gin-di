package apis

import (
	"github.com/gin-gonic/gin"
)

func Regester(route *gin.Engine) {
	// auth
	// authController := controllers.AuthController{}
	// route.POST("/login", authController.Login)
	// route.POST("/signup", authController.Signup)

	// // events
	// eventController := controllers.EventController{}
	// route.GET("/events", eventController.Index)
	// // route.POST("/events", middlewares.Authenticate, eventController.Create)
	// route.GET("/events/:id", eventController.Show)
	// // route.PUT("/events/:id", middlewares.Authenticate, eventController.Update)
	// // route.DELETE("/events/:id", middlewares.Authenticate, eventController.Delete)

	// // group it to middleware
	// auth := route.Group("/events")
	// auth.Use(middlewares.Authenticate)
	// auth.POST("", eventController.Create)
	// auth.PUT("/:id", eventController.Update)
	// auth.DELETE("/:id", eventController.Delete)
}
