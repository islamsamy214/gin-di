package fallbacks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"web-app/tests/support"
)

/*
 * These need no database: they assert on requests that never reach a handler.
 *
 * They are only meaningful because support.AppRouter builds the real engine —
 * NoRoute, NoMethod and HandleMethodNotAllowed are configured in
 * HTTPServiceProvider.Engine, so a hand-assembled test router would exercise
 * gin's defaults instead of this application's.
 */

// envelope is the four-key response shape every endpoint answers with.
type envelope struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    any                 `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

func decode(t *testing.T, body []byte) envelope {
	t.Helper()

	var parsed envelope
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decoding %s: %v", body, err)
	}

	return parsed
}

// Gin answers an unmatched route with the plain text "404 page not found",
// which breaks any client that assumes a JSON API replies in JSON.
func TestUnknownRouteReturnsTheJSONEnvelope(t *testing.T) {
	router, _ := support.AppRouter(t)

	res := support.GetPath(t, router, support.V1Path("/nope"), "")

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusNotFound, res.Body)
	}

	body := decode(t, res.Body.Bytes())

	if body.Status != "error" {
		t.Errorf("status = %q, want %q", body.Status, "error")
	}

	if body.Message == "" {
		t.Error("message is empty, want a client-facing summary")
	}
}

/*
 * A known path reached with the wrong verb must be 405, not 404.
 *
 * Gin only reaches NoMethod when HandleMethodNotAllowed is set, which it is not
 * by default — so this also pins that engine setting.
 */
func TestWrongMethodReturnsMethodNotAllowed(t *testing.T) {
	router, _ := support.AppRouter(t)

	res := support.Request(t, router, http.MethodDelete, support.V1Path("/events"), "", "")

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusMethodNotAllowed, res.Body)
	}

	// Gin fills this in itself, but only on the path that reaches NoMethod.
	if allow := res.Header().Get("Allow"); allow == "" {
		t.Error("Allow header is empty, want the methods the route accepts")
	}

	if body := decode(t, res.Body.Bytes()); body.Status != "error" {
		t.Errorf("status = %q, want %q", body.Status, "error")
	}
}

// Every response carries a correlation id, so a log line can be tied back to the
// request that produced it.
func TestResponsesEchoARequestID(t *testing.T) {
	router, _ := support.AppRouter(t)

	res := support.GetPath(t, router, "/", "")

	if id := res.Header().Get("X-Request-Id"); id == "" {
		t.Error("X-Request-Id is empty, want a generated correlation id")
	}
}

/*
 * An inbound request id is honoured so a trace survives an upstream hop, but only
 * when it is safe to write into a log line.
 *
 * The id is recorded on every log line for the request, so adopting an unchecked
 * caller-supplied value is a log-injection primitive: a newline in it forges
 * records. A rejected value must still yield an id, just a generated one.
 */
func TestRequestIDRejectsUnsafeInboundValues(t *testing.T) {
	router, _ := support.AppRouter(t)

	tests := []struct {
		name    string
		inbound string
		adopted bool
	}{
		{name: "clean value is adopted", inbound: "trace-abc_123.4", adopted: true},
		{name: "newline is rejected", inbound: "abc\ndef", adopted: false},
		{name: "space is rejected", inbound: "abc def", adopted: false},
		{name: "over-long is rejected", inbound: strings.Repeat("a", 65), adopted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			// Set directly on the map: http.Header.Set canonicalises and would
			// reject the control character before the middleware ever sees it.
			request.Header["X-Request-Id"] = []string{tt.inbound}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			echoed := response.Header().Get("X-Request-Id")
			if echoed == "" {
				t.Fatal("X-Request-Id is empty, want an id whether or not the inbound one was adopted")
			}

			if adopted := echoed == tt.inbound; adopted != tt.adopted {
				t.Errorf("echoed %q for inbound %q: adopted = %v, want %v", echoed, tt.inbound, adopted, tt.adopted)
			}
		})
	}
}
