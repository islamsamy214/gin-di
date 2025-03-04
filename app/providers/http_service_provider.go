package providers

import (
	"net/http"
	"time"
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

	// Add custom logger middleware
	router.Use(gin.LoggerWithWriter(configs.NewLogsConfig()))

	// Add custom recovery middleware
	router.Use(gin.Recovery())
}

func (r *HttpServiceProvider) Boot() {
	// Initialize gin engine
	r.init()

	// Create a new gin router
	router := gin.New()

	// Register the routes
	r.Register(router)

	// Add global middleware
	r.GlobalMiddleware(router)

	// Start the server
	(&http.Server{
		Addr:           configs.NewAppConfig().Host + ":" + configs.NewAppConfig().Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}).ListenAndServe()
}

func (r *HttpServiceProvider) init() {
	// Get the app config
	appCofing := configs.NewAppConfig()

	// Set gin mode
	if appCofing.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}
