package providers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"web-app/app/container"
	"web-app/app/exceptions"
	"web-app/app/http/middlewares"
	"web-app/app/http/responses"
	"web-app/app/services"
	"web-app/app/services/core"
	"web-app/configs"
	httpApis "web-app/routes/http"

	"github.com/gin-gonic/gin"
)

// Server timeouts. Every one of these was either absent or unset before, which
// left the process open to a client that connects and then simply waits.
const (
	// readHeaderTimeout bounds the request line and headers, which is the phase
	// a slowloris client stalls in.
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second

	// idleTimeout bounds a kept-alive connection between requests. Without it an
	// idle connection pins a goroutine indefinitely.
	idleTimeout = 60 * time.Second

	maxHeaderBytes = 1 << 20
)

// shutdownTimeout bounds the drain: in-flight requests get this long to finish
// before the listener is torn down regardless. It must stay below the
// supervisor's stopwaitsecs, or the process is killed mid-drain.
const shutdownTimeout = 15 * time.Second

// maxRequestBodyBytes is the largest request body any route accepts.
const maxRequestBodyBytes = 1 << 20

// maxMultipartMemory caps in-memory multipart buffering; gin's default is 32MB.
const maxMultipartMemory = 8 << 20

type HTTPServiceProvider struct{}

func NewHTTPServiceProvider() *HTTPServiceProvider {
	return &HTTPServiceProvider{}
}

/*
 * Boot resolves the application and serves it until signalled.
 *
 * Returns an error rather than calling log.Fatal so main owns the exit code, and
 * so the deferred cleanup below actually runs — log.Fatal skips deferred calls,
 * which would leak the database pool and the log file on every failure path.
 *
 * @return error The reason the server stopped, or nil on an orderly shutdown.
 */
func (provider *HTTPServiceProvider) Boot() error {
	// Resolved once. This used to be called three times per boot, twice inside
	// the server's Addr expression alone.
	appConfig := configs.NewAppConfig()

	provider.setMode(appConfig)

	logger, logFile, err := core.NewLogger(appConfig)
	if err != nil {
		return fmt.Errorf("boot: logger: %w", err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "closing log file: %v\n", err)
		}
	}()

	// Routes the log.Printf calls still in the migrator and console commands
	// through this handler, so they gain structure without being touched.
	slog.SetDefault(logger)

	jwtConfig, err := configs.NewJwtConfig()
	if err != nil {
		return fmt.Errorf("boot: jwt: %w", err)
	}

	db, err := core.Connection()
	if err != nil {
		return fmt.Errorf("boot: database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("closing database connection", slog.Any("error", err))
		}
	}()

	if err := NewValidationServiceProvider().Boot(); err != nil {
		return fmt.Errorf("boot: %w", err)
	}

	router, err := provider.Engine(container.New(container.Config{
		App:    appConfig,
		Auth:   services.NewAuthService(jwtConfig),
		Users:  services.NewUserService(db),
		DB:     db,
		Logger: logger,
	}))
	if err != nil {
		return fmt.Errorf("boot: %w", err)
	}

	return provider.serve(router, appConfig, logger)
}

/*
 * Engine builds the router.
 *
 * Separate from serving so tests can construct the real middleware stack and
 * route table without binding a port. It deliberately does not call
 * gin.SetMode: that is process-global, and a test that built the engine would
 * otherwise drag the application's mode along with it.
 *
 * @param c The resolved application.
 * @return *gin.Engine The configured router.
 * @return error       If the trusted proxy list is not valid.
 */
func (provider *HTTPServiceProvider) Engine(c *container.Container) (*gin.Engine, error) {
	router := gin.New()

	/*
	 * Gin's own documentation: it trusts every proxy unless told otherwise, and
	 * calls that "NOT safe". Untrusted, ClientIP() returns whatever the caller
	 * put in X-Forwarded-For — so the access log, and anything ever keyed on
	 * client address, is caller-controlled. An empty list means trust nothing,
	 * which is the correct default when there is no proxy in front.
	 */
	if err := router.SetTrustedProxies(c.App().TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}

	// Without this gin answers a wrong verb with 404 and NoMethod is never
	// reached. Gin fills in the Allow header itself.
	router.HandleMethodNotAllowed = true
	router.MaxMultipartMemory = maxMultipartMemory

	// Global middleware must be registered before the routes: gin snapshots the
	// handler chain when each route is registered, so anything added afterwards
	// is absent from routes already in the tree.
	provider.GlobalMiddleware(router, c)
	provider.Fallbacks(router)

	httpApis.Register(router, c)

	return router, nil
}

/*
 * GlobalMiddleware registers the middleware every route runs.
 *
 * The order is load-bearing, not alphabetical or arbitrary.
 *
 * @param router The engine to register on.
 * @param c      The resolved application.
 */
func (provider *HTTPServiceProvider) GlobalMiddleware(router *gin.Engine, c *container.Container) {
	router.Use(
		// First, so every log line and error below can correlate against it.
		middlewares.RequestID(),

		// Outside recovery, so it observes the 500 recovery produces rather than
		// missing the request entirely.
		middlewares.Logger(c.Logger()),

		middlewares.Recovery(c.Logger()),

		// Before any handler reads a body.
		middlewares.LimitBody(maxRequestBodyBytes),

		// Last, so it is the innermost wrapper and sees errors reported by every
		// route-level middleware and handler beneath it.
		middlewares.ExceptionHandler(c.Logger()),
	)
}

/*
 * Fallbacks answers requests that match no route.
 *
 * Gin's defaults return plain text here, which breaks a client that reasonably
 * assumes a JSON API answers in JSON.
 *
 * @param router The engine to register on.
 */
func (provider *HTTPServiceProvider) Fallbacks(router *gin.Engine) {
	router.NoRoute(func(ctx *gin.Context) {
		responses.Fail(ctx, http.StatusNotFound, exceptions.MessageNotFound, nil)
	})

	// Reached only because Engine sets HandleMethodNotAllowed.
	router.NoMethod(func(ctx *gin.Context) {
		responses.Fail(ctx, http.StatusMethodNotAllowed, exceptions.MessageNotAllowed, nil)
	})
}

/*
 * serve runs the HTTP server until a shutdown signal or a listener failure.
 *
 * @param router    The handler to serve.
 * @param appConfig Supplies the listen address.
 * @param logger    Where lifecycle events are recorded.
 * @return error The listener failure, or nil on an orderly shutdown.
 */
func (provider *HTTPServiceProvider) serve(router *gin.Engine, appConfig *configs.AppConfig, logger *slog.Logger) error {
	// Named and addressable, which is the entire point: the server used to be a
	// composite literal with ListenAndServe called directly on it, so there was
	// no value left to call Shutdown on.
	server := &http.Server{
		Addr:              net.JoinHostPort(appConfig.Host, appConfig.Port),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Buffered, so the goroutine can report and exit even if nothing is reading
	// yet. Reporting through a channel rather than calling log.Fatal from the
	// goroutine keeps the deferred cleanup in Boot reachable.
	listenErr := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}

		close(listenErr)
	}()

	logger.Info("http server listening",
		slog.String("addr", server.Addr),
		slog.String("mode", gin.Mode()),
	)

	select {
	case err, ok := <-listenErr:
		if ok && err != nil {
			return fmt.Errorf("http server: %w", err)
		}

		return nil
	case <-signalCtx.Done():
	}

	// Stop trapping signals before draining, so a second interrupt kills
	// immediately instead of being swallowed for the whole shutdown window.
	stop()

	logger.Info("shutting down", slog.Duration("timeout", shutdownTimeout))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("forced shutdown after %s: %w", shutdownTimeout, err)
	}

	logger.Info("shutdown complete")

	return nil
}

/*
 * setMode selects gin's operating mode.
 *
 * @param appConfig Supplies the debug flag.
 */
func (provider *HTTPServiceProvider) setMode(appConfig *configs.AppConfig) {
	if appConfig.Debug {
		gin.SetMode(gin.DebugMode)

		return
	}

	gin.SetMode(gin.ReleaseMode)
}
