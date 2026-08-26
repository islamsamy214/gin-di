package middlewares

import (
	"errors"
	"log/slog"
	"net/http"
	"web-app/app/exceptions"
	"web-app/app/http/responses"

	"github.com/gin-gonic/gin"
)

/*
 * ExceptionHandler renders whatever handlers reported with ctx.Error().
 *
 * This is Laravel's Exception Handler. A handler's job is to report a failure
 * and return; the status code, the body shape and the logging all happen here,
 * exactly once. It replaces thirteen ad-hoc ctx.JSON sites that disagreed about
 * all three, several of which returned raw driver text to the caller.
 *
 * @param logger Where failures are recorded.
 */
func ExceptionHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		// The last reported error decides the response. Earlier ones stay in
		// ctx.Errors and are still logged below, so nothing is silently dropped.
		reported := ctx.Errors.Last().Err

		exception, ok := errors.AsType[*exceptions.HTTPException](reported)
		if !ok {
			// An error that reached here without being classified is a bug in the
			// handler, not something the caller can act on. Treat it as internal
			// and let the log carry the detail.
			exception = exceptions.NewInternal(reported)
		}

		level := slog.LevelWarn
		if exception.Status >= http.StatusInternalServerError {
			level = slog.LevelError
		}

		// The cause is logged here and only here. Note it is logged even when the
		// response has already been written, because the failure happened either
		// way.
		logger.Log(ctx.Request.Context(), level, "request failed",
			slog.String("request_id", RequestIDFrom(ctx)),
			slog.Int("status", exception.Status),
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", reported),
		)

		// A handler that already streamed a response wins: appending a second
		// body to the same response would corrupt it.
		if ctx.Writer.Written() {
			return
		}

		responses.Fail(ctx, exception.Status, exception.Message, exception.Errors)
	}
}
