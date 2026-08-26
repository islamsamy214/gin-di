package providers

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// DateLayout is the wire format for date-only fields, enforced by the eventdate
// rule below and by the database column behind them.
const DateLayout = "2006-01-02"

// jsonTagSeparator splits a json struct tag from its options (",omitempty").
const jsonTagSeparator = ","

type ValidationServiceProvider struct{}

func NewValidationServiceProvider() *ValidationServiceProvider {
	return &ValidationServiceProvider{}
}

/*
 * Boot registers this application's validation rules on gin's validator.
 *
 * Called before the engine is built. Gin resolves binding.Validator lazily and
 * shares one instance process-wide, so registering here applies to every
 * subsequent bind without threading anything through.
 *
 * @return error If gin's validator is not the engine this code expects, or a
 *               rule could not be registered.
 */
func (provider *ValidationServiceProvider) Boot() error {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		// Fail loudly rather than silently skipping registration: a missing
		// eventdate rule makes validator treat the tag as unknown and panic on
		// the first bind, far from the cause.
		return errors.New("validation: gin's validator engine is not *validator.Validate")
	}

	provider.registerFieldNames(engine)

	if err := engine.RegisterValidation("eventdate", validDate); err != nil {
		return fmt.Errorf("validation: registering eventdate: %w", err)
	}

	return nil
}

/*
 * registerFieldNames makes validation errors report json field names.
 *
 * Without this a failure on LoginRequest.Username is reported as
 * "LoginRequest.Username", which no client can map back to the field it sent.
 * With it the key is "username", so the errors map in the response envelope is
 * directly usable.
 */
func (provider *ValidationServiceProvider) registerFieldNames(engine *validator.Validate) {
	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), jsonTagSeparator, 2)[0]

		switch name {
		case "-":
			// Not serialized, so it has no wire name to report.
			return ""
		case "":
			// Bound from a form or query tag instead; fall back to the Go name.
			return field.Name
		default:
			return name
		}
	})
}

/*
 * validDate reports whether a field holds a date the database will accept.
 *
 * `required` alone accepts any non-empty string, which then reaches Postgres and
 * returns as a driver error rendered at HTTP 500. A malformed date is invalid
 * input and belongs in a 422, so it has to be rejected here.
 */
func validDate(field validator.FieldLevel) bool {
	_, err := time.Parse(DateLayout, field.Field().String())

	return err == nil
}
