package providers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type AppServiceProvider struct {
	//
}

func NewAppServiceProvider() *AppServiceProvider {
	return &AppServiceProvider{}
}

func (s *AppServiceProvider) Register() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Register the app service provider middleware "As a global middleware"
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
