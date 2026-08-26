package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"web-app/app/exceptions"
	"web-app/app/http/responses"

	"github.com/gin-gonic/gin"
)

/*
 * Recovery renders a panic as the standard error envelope.
 *
 * gin.Recovery alone writes a bare 500 with no body, so a client that panicked
 * the server gets a response it cannot decode while the envelope holds
 * everywhere else. gin.CustomRecovery lets the same shape apply to the one case
 * no handler got to answer.
 *
 * @param logger Where the panic and its stack are recorded.
 */
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered any) {
		// The stack is captured once, here. It is the only record of where the
		// panic came from, so it is logged at Error regardless of configuration.
		logger.Error("panic recovered",
			slog.Any("panic", recovered),
			slog.String("request_id", RequestIDFrom(ctx)),
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.String("stack", string(debug.Stack())),
		)

		if ctx.Writer.Written() {
			// A partially written response cannot be turned into a clean
			// envelope; abort so nothing further is appended to it.
			ctx.Abort()

			return
		}

		responses.Fail(ctx, http.StatusInternalServerError, exceptions.MessageInternal, nil)
	})
}
