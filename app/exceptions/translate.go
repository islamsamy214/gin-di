package exceptions

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

// SQLSTATE classes, per the PostgreSQL error-code appendix. Only the class is
// consulted; see FromDatabaseError for why the message never is.
const (
	sqlStateClassDataException      = "22"
	sqlStateClassIntegrityViolation = "23"
)

/*
 * FromBindError turns whatever ShouldBind returned into a renderable exception.
 *
 * Three distinct outcomes, because they are three distinct client mistakes:
 * a body that violates the rules is a 422 with per-field detail, a body that is
 * not valid JSON at all is a 400 (nothing was validated, so there are no fields
 * to report), and anything else is a 400 with no detail rather than a guess.
 *
 * @param err The error returned by a ShouldBind call.
 * @return *HTTPException Never nil.
 */
func FromBindError(err error) *HTTPException {
	if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
		fields := make(map[string][]string, len(validationErrors))

		for _, fieldError := range validationErrors {
			// Field() is the json name rather than the Go field name, because
			// ValidationServiceProvider registers a tag-name function. Without
			// that, these keys would read "LoginRequest.Username" and no client
			// could map them back to its form.
			name := fieldError.Field()
			fields[name] = append(fields[name], describe(fieldError))
		}

		return NewValidation(fields, err)
	}

	// An empty body reaches here as io.EOF, and a truncated one as
	// io.ErrUnexpectedEOF.
	_, syntaxError := errors.AsType[*json.SyntaxError](err)
	_, typeError := errors.AsType[*json.UnmarshalTypeError](err)

	if syntaxError || typeError || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return NewBadRequest("Malformed JSON body", err)
	}

	return NewBadRequest(MessageBadRequest, err)
}

/*
 * describe renders one field failure as a client-facing sentence.
 *
 * Built from the tag and its parameter rather than from the validator's own
 * Error() output, which embeds the Go struct namespace and reads as an internal
 * diagnostic.
 */
func describe(fieldError validator.FieldError) string {
	field := fieldError.Field()

	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fieldError.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "eventdate":
		// Spelled out rather than referencing the Go layout constant: a client
		// reading this message has no idea what "2006-01-02" means.
		return fmt.Sprintf("%s must be a valid date in YYYY-MM-DD format", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

/*
 * FromDatabaseError classifies a data-layer failure.
 *
 * A *pq.Error is deliberately never inspected for its message: that text
 * carries table, column and constraint names, and returning it verbatim — which
 * the events controller used to do at HTTP 500 — hands an attacker the schema.
 * Only the SQLSTATE class is used, and anything unrecognised becomes a generic
 * internal error.
 *
 * @param err The error returned by a model or service.
 * @return *HTTPException Never nil.
 */
func FromDatabaseError(err error) *HTTPException {
	if errors.Is(err, sql.ErrNoRows) {
		return NewNotFound(err)
	}

	if pqError, ok := errors.AsType[*pq.Error](err); ok {
		switch pqError.Code.Class() {
		case sqlStateClassIntegrityViolation:
			// Unique violation, foreign key violation, not-null violation: all
			// are the caller asking for something the current state forbids.
			return NewConflict(err)
		case sqlStateClassDataException:
			// A value the column cannot hold — an unparseable date, say. That is
			// invalid input, not a server fault, so it must not be a 500.
			return NewValidation(nil, err)
		}
	}

	return NewInternal(err)
}

/*
 * FieldErrors reports a single field failure without going through a validator.
 *
 * For rules a struct tag cannot express — a cross-field constraint, or one that
 * needs a database lookup — so those failures still render in the same shape as
 * tag-driven ones.
 */
func FieldErrors(field, message string) *HTTPException {
	return NewValidation(map[string][]string{field: {message}}, errors.New(strings.ToLower(message)))
}
