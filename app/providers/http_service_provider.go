package providers

import (
	"web-app/configs"
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
	server := configs.NewHttpServer(router)
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
