package providers

import (
	"net/http"
	"os"
	"time"
	"web-app/app/services"
	"web-app/app/services/core"

	"github.com/gin-gonic/gin"
)

type AppServiceProvider struct {
	container *core.Container
}

func NewAppServiceProvider() *AppServiceProvider {
	container := core.NewContainer()

	// Register services here
	container.Bind("MyService", services.NewService())

	return &AppServiceProvider{container: container}
}

func (s *AppServiceProvider) Register() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Set the container in the request context
		ctx.Set("container", s.container)
		ctx.Next() // Continue to the next middleware/handler
	}
}

func (s *AppServiceProvider) Boot(router *gin.Engine) {
	// Boot the serve service
	server := &http.Server{
		Addr:           "0.0.0.0:" + os.Getenv("APP_PORT"),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	server.ListenAndServe()
}
