package providers

import (
	web "web-app/routes/http"
	api "web-app/routes/http/apis"

	"github.com/gin-gonic/gin"
)

type RouteServiceProvider struct{}

func (r *RouteServiceProvider) Boot(router *gin.Engine) {
	// Boot the route service
	web.Regester(router)
	api.Regester(router)
}
