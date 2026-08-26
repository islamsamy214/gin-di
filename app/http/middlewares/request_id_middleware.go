package middlewares

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader is the header a correlation id is read from and echoed back on.
const RequestIDHeader = "X-Request-Id"

// contextRequestID is the gin context key the id is stored under.
const contextRequestID = "requestId"

// requestIDBytes is the length of a generated id before hex encoding.
const requestIDBytes = 16

// maxRequestIDLength bounds an id accepted from a caller.
const maxRequestIDLength = 64

/*
 * RequestID attaches a correlation id to the request and echoes it back.
 *
 * An inbound header is honoured so a trace survives an upstream hop, but it is
 * validated first. The id is written into every log line for the request, so an
 * unchecked caller-supplied value is a log-injection primitive: a newline in it
 * forges log records.
 *
 * @return gin.HandlerFunc Registered first, so everything downstream can
 *                         correlate against the id it sets.
 */
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetHeader(RequestIDHeader)

		if !validRequestID(id) {
			id = newRequestID()
		}

		ctx.Set(contextRequestID, id)
		ctx.Writer.Header().Set(RequestIDHeader, id)

		ctx.Next()
	}
}

/*
 * RequestIDFrom reads the id back off a request.
 *
 * @param ctx The request to read.
 * @return string The id, empty when the middleware did not run.
 */
func RequestIDFrom(ctx *gin.Context) string {
	return ctx.GetString(contextRequestID)
}

/*
 * validRequestID reports whether a caller-supplied id is safe to adopt.
 *
 * An allowlist rather than a denylist: the set of characters that are harmless
 * in a log field is small and known, whereas the set that causes trouble
 * somewhere downstream is open-ended.
 */
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}

	for _, character := range id {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '-', character == '_', character == '.':
		default:
			return false
		}
	}

	return true
}

/*
 * newRequestID generates a fresh id.
 *
 * crypto/rand rather than a counter or a timestamp, so ids stay unique across
 * replicas and restarts without any coordination. rand.Read never returns an
 * error on any supported platform, hence no error path here.
 */
func newRequestID() string {
	buffer := make([]byte, requestIDBytes)
	_, _ = rand.Read(buffer)

	return hex.EncodeToString(buffer)
}
