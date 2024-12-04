package routes

import (
	"net/http"

	"web-app/app/services"
	"web-app/app/services/core"

	"github.com/gin-gonic/gin"
)

func Regester(route *gin.Engine) {
	route.GET("/", func(ctx *gin.Context) {
		container := ctx.MustGet("container").(*core.Container) // Import core package

		service := container.Resolve("MyService") // Import services package
		myService := service.(*services.MyService)

		message := myService.GetHello()
		ctx.JSON(http.StatusOK, gin.H{"message": message})
	})
}
