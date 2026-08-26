/*
 * Package responses owns the wire format for every JSON body this application
 * sends, mirroring the successResponse()/failResponse() helpers of the Laravel
 * side of the house.
 *
 * Handlers call Success or Fail and nothing else. Before this the codebase had
 * thirteen hand-built gin.H literals across five files emitting four mutually
 * incompatible shapes, so a client could not write one decoder for the API.
 */
package responses

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// The status discriminator. A string rather than a boolean success flag, so a
// third state can be added without breaking existing decoders.
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// DefaultMessage is used when a handler supplies no message of its own.
const DefaultMessage = "ok"

/*
 * Envelope is the shape of every response body.
 *
 * All four keys are always present. There is deliberately no omitempty: a nil
 * `any` and a nil map both marshal to null, which is the contract — a client
 * can read response.errors without first checking whether the key exists. The
 * field order here is the key order on the wire.
 */
type Envelope struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    any                 `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

/*
 * Success writes a successful response.
 *
 * @param ctx     The request being answered.
 * @param status  The HTTP status to send.
 * @param message A short sentence describing what happened; DefaultMessage is
 *                used when empty.
 * @param data    The payload, which must already be a resource rather than a
 *                model — see app/http/resources.
 */
func Success(ctx *gin.Context, status int, message string, data any) {
	if message == "" {
		message = DefaultMessage
	}

	// Note on SecureJSON: gin only prepends its while(1); guard when the
	// marshalled value is a top-level array. Under this envelope the top level
	// is always an object, so enabling it would be a guaranteed no-op. The
	// envelope is itself the JSON-hijacking defence.
	ctx.JSON(status, Envelope{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
		Errors:  nil,
	})
}

/*
 * Fail writes an error response and aborts the handler chain.
 *
 * Aborting is part of the contract: a rendered failure must be the last word on
 * the request, or a downstream handler appends a second body to the same
 * response.
 *
 * @param ctx     The request being answered.
 * @param status  The HTTP status to send.
 * @param message A safe, client-facing summary. Never pass an error's own text:
 *                driver and parser messages carry table names, constraint names
 *                and parser internals.
 * @param errors  Field-level detail, or nil when the failure is not field-level.
 */
func Fail(ctx *gin.Context, status int, message string, errors map[string][]string) {
	if message == "" {
		message = http.StatusText(status)
	}

	ctx.AbortWithStatusJSON(status, Envelope{
		Status:  StatusError,
		Message: message,
		Data:    nil,
		Errors:  errors,
	})
}
