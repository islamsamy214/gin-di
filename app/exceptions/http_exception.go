/*
 * Package exceptions holds the error types the HTTP layer knows how to render,
 * mirroring Laravel's app/Exceptions.
 *
 * A handler reports one of these with ctx.Error() and returns; the status code,
 * body shape and logging all happen once, in the exception handler middleware.
 */
package exceptions

import "net/http"

// Client-facing messages. Fixed strings, deliberately: the alternative is
// interpolating an error's own text, which is how table names, constraint names
// and parser internals reach a caller.
const (
	MessageBadRequest   = "The request could not be read"
	MessageUnauthorized = "Unauthorized"
	MessageForbidden    = "Forbidden"
	MessageNotFound     = "Not found"
	MessageNotAllowed   = "Method not allowed"
	MessageConflict     = "The request conflicts with existing data"
	MessageValidation   = "The given data was invalid."
	MessageTooMany      = "Too many requests"
	MessageInternal     = "Something went wrong"
)

/*
 * HTTPException is an error that knows how it should be rendered.
 *
 * Err is the cause. It is logged and never sent: that split is the whole point
 * of the type, and is what separates the diagnostic detail an operator needs
 * from the information a caller is allowed to learn.
 */
type HTTPException struct {
	Status  int
	Message string
	Errors  map[string][]string
	Err     error
}

// Error satisfies the error interface with the client-facing message, so an
// exception that is accidentally logged directly still cannot leak the cause.
func (exception *HTTPException) Error() string {
	return exception.Message
}

// Unwrap exposes the cause to errors.Is and errors.As.
func (exception *HTTPException) Unwrap() error {
	return exception.Err
}

// NewBadRequest reports a request that could not be read at all.
func NewBadRequest(message string, cause error) *HTTPException {
	if message == "" {
		message = MessageBadRequest
	}

	return &HTTPException{Status: http.StatusBadRequest, Message: message, Err: cause}
}

// NewUnauthorized reports missing or unusable credentials.
func NewUnauthorized() *HTTPException {
	return &HTTPException{Status: http.StatusUnauthorized, Message: MessageUnauthorized}
}

// NewForbidden reports an authenticated caller acting outside its permissions.
func NewForbidden() *HTTPException {
	return &HTTPException{Status: http.StatusForbidden, Message: MessageForbidden}
}

// NewNotFound reports an absent resource.
func NewNotFound(cause error) *HTTPException {
	return &HTTPException{Status: http.StatusNotFound, Message: MessageNotFound, Err: cause}
}

// NewMethodNotAllowed reports a known path reached with the wrong verb.
func NewMethodNotAllowed() *HTTPException {
	return &HTTPException{Status: http.StatusMethodNotAllowed, Message: MessageNotAllowed}
}

// NewConflict reports a request the current state cannot accept.
func NewConflict(cause error) *HTTPException {
	return &HTTPException{Status: http.StatusConflict, Message: MessageConflict, Err: cause}
}

// NewValidation reports field-level validation failures.
func NewValidation(fields map[string][]string, cause error) *HTTPException {
	return &HTTPException{
		Status:  http.StatusUnprocessableEntity,
		Message: MessageValidation,
		Errors:  fields,
		Err:     cause,
	}
}

/*
 * NewTooManyRequests reports an exhausted rate limit.
 *
 * Parameterless like NewUnauthorized: how long to wait is a response header, not
 * part of the body, so the throttle middleware sets it directly rather than
 * carrying it through the exception. Widening this type with an HTTP header
 * would put transport detail in the error vocabulary.
 */
func NewTooManyRequests() *HTTPException {
	return &HTTPException{Status: http.StatusTooManyRequests, Message: MessageTooMany}
}

/*
 * NewInternal reports a failure the caller can do nothing about.
 *
 * The message is fixed and the cause is carried separately, so the only way to
 * learn what actually broke is to read the log.
 */
func NewInternal(cause error) *HTTPException {
	return &HTTPException{Status: http.StatusInternalServerError, Message: MessageInternal, Err: cause}
}
