package providers

import (
	"net/http"
	"os"
	"time"
	web "web-app/routes/http"
	api "web-app/routes/http/apis"

	"github.com/gin-gonic/gin"
)

type HttpServiceProvider struct{}

func NewHttpServiceProvider() *HttpServiceProvider {
	return &HttpServiceProvider{}
}

func (r *HttpServiceProvider) Register(router *gin.Engine) {
	// Register the routes
	web.Regester(router)
	api.Regester(router)
}

func (r *HttpServiceProvider) GlobalMiddleware(router *gin.Engine) {
	// Add global middleware here
}

func (r *HttpServiceProvider) Serve(router *gin.Engine) {
	server := &http.Server{
		Addr:           "0.0.0.0:" + os.Getenv("APP_PORT"),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	server.ListenAndServe()
}

func (r *HttpServiceProvider) Boot() {
	// Create a new gin router
	router := gin.Default()

	// Register the routes
	r.Register(router)

	// Add global middleware
	r.GlobalMiddleware(router)

	// Start the server
	r.Serve(router)
}
