package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
 * LimitBody caps how much of a request body a handler can read.
 *
 * Without it ShouldBindJSON reads an arbitrarily large body into memory before
 * deciding it is invalid. On the login route each of those bodies also buys the
 * sender an argon2id verification, so the body limit and the verification
 * semaphore in UserService bound the same attack from two directions.
 *
 * MaxBytesReader is applied to the body rather than checking Content-Length: a
 * caller controls that header, and a chunked request has none at all.
 *
 * @param maxBytes The largest body accepted, in bytes.
 */
func LimitBody(maxBytes int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)

		ctx.Next()
	}
}
