package configs

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func NewHttpServer(router *gin.Engine) *http.Server {
	appConfig := NewAppConfig()
	return &http.Server{
		Addr:           appConfig.Host + ":" + appConfig.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
