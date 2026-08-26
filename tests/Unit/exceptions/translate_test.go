package unit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"web-app/app/exceptions"

	"github.com/lib/pq"
)

/*
 * The translation layer is the only thing standing between a driver error and a
 * client, so these tests assert two properties on every case: the status is the
 * one a caller should see, and the rendered message does not contain any text
 * from the underlying cause.
 *
 * No database required — every input is synthesised.
 */

func TestFromDatabaseErrorClassifiesByCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "no rows is a 404",
			err:        sql.ErrNoRows,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unique violation is a 409",
			err:        &pq.Error{Code: "23505", Message: `duplicate key value violates unique constraint "users_username_key"`},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "foreign key violation is a 409",
			err:        &pq.Error{Code: "23503", Message: `insert or update on table "events" violates foreign key constraint "events_user_id_fkey"`},
			wantStatus: http.StatusConflict,
		},
		{
			// The live bug this fixes: a date string that passes `required` but
			// Postgres cannot parse used to surface as a 500 carrying the raw
			// driver message.
			name:       "invalid date syntax is a 422",
			err:        &pq.Error{Code: "22007", Message: `invalid input syntax for type date: "not-a-date"`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "an unclassified error is a 500",
			err:        errors.New("connection reset by peer"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exception := exceptions.FromDatabaseError(tt.err)

			if exception.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", exception.Status, tt.wantStatus)
			}

			assertMessageHidesCause(t, exception.Message, tt.err)

			// The cause must remain reachable for the log even though it is not
			// rendered.
			if !errors.Is(exception, tt.err) {
				t.Errorf("errors.Is(exception, cause) = false, want the cause to stay wrapped")
			}
		})
	}
}

func TestFromBindErrorClassifiesByKind(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "malformed json is a 400",
			err:        &json.SyntaxError{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong type is a 400",
			err:        &json.UnmarshalTypeError{Value: "string", Field: "page"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "an empty body is a 400",
			err:        io.EOF,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "an unrecognised bind failure is a 400",
			err:        errors.New("some other bind problem"),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exception := exceptions.FromBindError(tt.err)

			if exception.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", exception.Status, tt.wantStatus)
			}

			// A body that could not be parsed produced no field failures, so
			// there is nothing per-field to report.
			if len(exception.Errors) != 0 {
				t.Errorf("Errors = %v, want none for a body that never validated", exception.Errors)
			}
		})
	}
}

// A 500's message must be fixed, so no internal detail can ride out on it.
func TestInternalMessageIsFixed(t *testing.T) {
	cause := errors.New("pq: password authentication failed for user \"root\"")

	exception := exceptions.NewInternal(cause)

	if exception.Message != exceptions.MessageInternal {
		t.Errorf("Message = %q, want the fixed %q", exception.Message, exceptions.MessageInternal)
	}

	assertMessageHidesCause(t, exception.Message, cause)
}

/*
 * safeMessages is every message the translation layer is allowed to render.
 *
 * A closed allowlist is the real property under test, and a stronger one than
 * diffing the message against the cause: a generic message and a driver string
 * can share ordinary English words ("invalid") without anything having leaked,
 * so word comparison produces false positives while still not proving the
 * message was never interpolated. Membership here proves exactly that.
 */
var safeMessages = map[string]bool{
	exceptions.MessageBadRequest:   true,
	exceptions.MessageUnauthorized: true,
	exceptions.MessageForbidden:    true,
	exceptions.MessageNotFound:     true,
	exceptions.MessageNotAllowed:   true,
	exceptions.MessageConflict:     true,
	exceptions.MessageValidation:   true,
	exceptions.MessageInternal:     true,
	"Malformed JSON body":          true,
}

/*
 * assertMessageHidesCause fails if the rendered message is not a fixed, known
 * string.
 *
 * Any message outside the allowlist has been built from something — and the only
 * thing available to build it from is the cause.
 */
func assertMessageHidesCause(t *testing.T, message string, cause error) {
	t.Helper()

	if !safeMessages[message] {
		t.Errorf("message %q is not one of the fixed safe messages; it appears derived from the cause %q", message, cause)
	}

	// Belt and braces on the specific shape that matters most: driver errors
	// name tables, columns and constraints inside double quotes.
	if strings.Contains(message, `"`) {
		t.Errorf("message %q contains a quoted identifier, which is how schema names escape", message)
	}
}
