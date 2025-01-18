package routes

import (
	"net/http"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

func Regester(route *gin.Engine) {
	route.GET("/", func(ctx *gin.Context) {

		myService := services.NewMyService()

		message := myService.GetHello()
		ctx.JSON(http.StatusOK, gin.H{"message": message})
	})
}
