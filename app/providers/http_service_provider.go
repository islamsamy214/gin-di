package providers

import (
	"errors"
	"log"
	"net/http"
	"time"
	"web-app/app/http/middlewares"
	"web-app/app/services"
	"web-app/configs"
	httpApis "web-app/routes/http"

	"github.com/gin-gonic/gin"
)

type HttpServiceProvider struct{}

func NewHttpServiceProvider() *HttpServiceProvider {
	return &HttpServiceProvider{}
}

func (provider *HttpServiceProvider) Boot() {
	// Initialize gin engine
	provider.init()

	// Composition root: config is resolved once here, then injected downward.
	jwtConfig, err := configs.NewJwtConfig()
	if err != nil {
		log.Fatalf("boot: %v", err)
	}

	authService := services.NewAuthService(jwtConfig)

	// Create a new gin router
	router := gin.New()

	// Global middleware must be registered before the routes: gin snapshots the
	// handler chain at registration time, so anything added afterwards is
	provider.GlobalMiddleware(router)

	// Register the routes
	provider.Register(router, authService)

	// Start the server
	serverErr := (&http.Server{
		Addr:           configs.NewAppConfig().Host + ":" + configs.NewAppConfig().Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}).ListenAndServe()

	// A closed server is an orderly shutdown; anything else must not exit 0.
	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Fatalf("http server: %v", serverErr)
	}
}

func (provider *HttpServiceProvider) Register(router *gin.Engine, auth *services.AuthService) {
	// Register the routes
	httpApis.Register(router, auth)
}

func (provider *HttpServiceProvider) GlobalMiddleware(router *gin.Engine) {
	// Add global middleware here
	router.Use(gin.LoggerWithWriter(middlewares.NewLogIOWriterMiddleware()))

	// Add custom recovery middleware
	router.Use(gin.Recovery())
}

func (provider *HttpServiceProvider) init() {
	// Get the app config
	appCofing := configs.NewAppConfig()

	// Set gin mode
	if appCofing.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}
