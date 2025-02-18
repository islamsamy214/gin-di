package configs

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func NewHttpServer(router *gin.Engine) *http.Server {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8000"
	}

	return &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
