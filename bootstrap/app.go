package bootstrap

import (
	"web-app/app/providers"

	"github.com/gin-gonic/gin"
)

func Boot() {
	// Load the .env file
	(&providers.EnvServiceProvider{}).Boot()

	// Gin's default logger to use our custom writer
	(&providers.LogServiceProvider{}).Boot()

	// Create a new gin router
	router := gin.Default()

	// Create and register the app service provider
	appServiceProvider := providers.NewAppServiceProvider()
	router.Use(appServiceProvider.Register())

	// Register the routes
	(&providers.RouteServiceProvider{}).Boot(router)

	// Start the server
	appServiceProvider.Boot(router)
}
