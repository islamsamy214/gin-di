package middlewares

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

/*
 * Logger records one structured line per request.
 *
 * This replaces gin.LoggerWithWriter and the io.Writer factory that used to
 * stand in for a middleware here. That factory reconfigured the global log
 * package as a side effect of being constructed, fixed its filename at boot so
 * a long-running process never rolled over to a new day, never closed the file,
 * and swept old logs from a detached goroutine whose failures nothing could
 * observe. Log rotation now belongs to core.NewLogger; this file logs requests.
 *
 * @param logger The application logger.
 */
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		logger.LogAttrs(ctx.Request.Context(), slog.LevelInfo, "request",
			slog.String("request_id", RequestIDFrom(ctx)),
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.Int("status", ctx.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.Int("bytes", ctx.Writer.Size()),

			// Trustworthy only because the engine sets trusted proxies. With
			// gin's default of trusting every proxy, this field is whatever the
			// caller put in X-Forwarded-For.
			slog.String("ip", ctx.ClientIP()),
		)
	}
}
