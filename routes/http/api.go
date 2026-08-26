package http

import (
	"net/http"
	"web-app/app/container"
	"web-app/app/http/responses"
	v1 "web-app/routes/http/v1"

	"github.com/gin-gonic/gin"
)

// APIPrefix is the mount point every versioned API group hangs from.
const APIPrefix = "/api"

/*
 * Register wires the unversioned surface and mounts each API version under its
 * own group.
 *
 * Adding a version means one Group plus one Register call here; no existing
 * version is touched.
 *
 * @param router The engine to register on.
 * @param c      The resolved application, passed down into each version.
 */
func Register(router *gin.Engine, c *container.Container) {
	// home route, deliberately outside the versioned API
	router.GET("/", welcome)

	api := router.Group(APIPrefix)

	v1.Register(api.Group("/v1"), c)
}

func welcome(ctx *gin.Context) {
	responses.Success(ctx, http.StatusOK, "Welcome to the home page", nil)
}
