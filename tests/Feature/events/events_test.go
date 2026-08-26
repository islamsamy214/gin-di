package events

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"web-app/app/http/middlewares"
	"web-app/app/models"
	"web-app/database/factories"
	"web-app/tests/support"
)

/*
 * eventsFrom decodes the events endpoint's response.
 *
 * The rows moved from data to data.events when pagination metadata was added
 * alongside them, which keeps the envelope itself exactly four keys wide.
 */
func eventsFrom(t *testing.T, body []byte) []models.Event {
	t.Helper()

	var parsed struct {
		Data struct {
			Events []models.Event `json:"events"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decoding events response %s: %v", body, err)
	}

	return parsed.Data.Events
}

// eventFrom decodes the single-event payload a create returns.
func eventFrom(t *testing.T, body []byte) models.Event {
	t.Helper()

	var parsed struct {
		Data struct {
			Event models.Event `json:"event"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decoding event response %s: %v", body, err)
	}

	return parsed.Data.Event
}

func TestEventsReturnsFactoryEvents(t *testing.T) {
	support.FreshDatabase(t)

	router, auth := support.AppRouter(t)

	owner, err := factories.UserFactory().State(factories.WithUsername("dave")).CreateOne()
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}

	created, err := factories.EventFactory().Count(3).State(factories.ForUser(owner)).Create()
	if err != nil {
		t.Fatalf("creating events: %v", err)
	}

	token, err := auth.GenerateToken(owner.ID, owner.Username)
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	res := support.GetPath(t, router, support.V1Path("/events"), middlewares.BearerPrefix+token)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	events := eventsFrom(t, res.Body.Bytes())

	if len(events) != len(created) {
		t.Fatalf("returned %d events, want %d (body: %s)", len(events), len(created), res.Body)
	}

	// Every returned event must belong to the owner the factory stated.
	names := make(map[string]struct{}, len(events))

	for _, event := range events {
		if event.UserID != owner.ID {
			t.Errorf("event %q has user_id %d, want %d", event.Name, event.UserID, owner.ID)
		}

		if event.ID == 0 {
			t.Errorf("event %q came back with id 0", event.Name)
		}

		names[event.Name] = struct{}{}
	}

	for _, event := range created {
		if _, found := names[event.Name]; !found {
			t.Errorf("created event %q is missing from the response", event.Name)
		}
	}
}

/*
 * Regression: Index called the unscoped Paginate, so any authenticated caller
 * read every user's events.
 *
 * The existing coverage could not catch this — a fresh database held exactly one
 * user, so an unscoped read and a correctly scoped one return the same rows. Two
 * owners with different event counts is the minimum needed to tell them apart.
 */
func TestEventsOnlyReturnsTheCallersEvents(t *testing.T) {
	support.FreshDatabase(t)

	router, auth := support.AppRouter(t)

	mine, err := factories.UserFactory().State(factories.WithUsername("grace")).CreateOne()
	if err != nil {
		t.Fatalf("creating the caller: %v", err)
	}

	theirs, err := factories.UserFactory().State(factories.WithUsername("heidi")).CreateOne()
	if err != nil {
		t.Fatalf("creating the other user: %v", err)
	}

	if _, err := factories.EventFactory().Count(2).State(factories.ForUser(mine)).Create(); err != nil {
		t.Fatalf("creating the caller's events: %v", err)
	}

	if _, err := factories.EventFactory().Count(3).State(factories.ForUser(theirs)).Create(); err != nil {
		t.Fatalf("creating the other user's events: %v", err)
	}

	token, err := auth.GenerateToken(mine.ID, mine.Username)
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	res := support.GetPath(t, router, support.V1Path("/events"), middlewares.BearerPrefix+token)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	events := eventsFrom(t, res.Body.Bytes())

	if len(events) != 2 {
		t.Fatalf("returned %d events, want 2 — the read is not scoped to the caller (body: %s)", len(events), res.Body)
	}

	for _, event := range events {
		if event.UserID != mine.ID {
			t.Errorf("event %q belongs to user %d, not the caller %d", event.Name, event.UserID, mine.ID)
		}
	}
}

/*
 * Regression: Create hardcoded UserID = 1, ignoring the authenticated identity
 * the middleware had already resolved.
 *
 * Two throwaway users are created first so the real owner cannot land on id 1 by
 * coincidence — without that, the test would pass against the bug.
 */
func TestCreateEventAttachesTheAuthenticatedUser(t *testing.T) {
	support.FreshDatabase(t)

	router, auth := support.AppRouter(t)

	if _, err := factories.UserFactory().Count(2).Create(); err != nil {
		t.Fatalf("creating the padding users: %v", err)
	}

	owner, err := factories.UserFactory().State(factories.WithUsername("frank")).CreateOne()
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}

	if owner.ID == 1 {
		t.Fatal("owner landed on id 1, so this test cannot distinguish the hardcoded value")
	}

	token, err := auth.GenerateToken(owner.ID, owner.Username)
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	res := support.PostJSONAs(t, router, support.V1Path("/events"),
		`{"name":"standup","date":"2030-01-01"}`, middlewares.BearerPrefix+token)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusCreated, res.Body)
	}

	created := eventFrom(t, res.Body.Bytes())

	if created.UserID != owner.ID {
		t.Errorf("created event user_id = %d, want the authenticated %d (body: %s)", created.UserID, owner.ID, res.Body)
	}
}

// A malformed date must be rejected at validation, not passed to Postgres and
// returned as a driver error at HTTP 500.
func TestCreateEventRejectsAMalformedDate(t *testing.T) {
	support.FreshDatabase(t)

	router, auth := support.AppRouter(t)

	owner, err := factories.UserFactory().CreateOne()
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}

	token, err := auth.GenerateToken(owner.ID, owner.Username)
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	res := support.PostJSONAs(t, router, support.V1Path("/events"),
		`{"name":"standup","date":"not-a-date"}`, middlewares.BearerPrefix+token)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusUnprocessableEntity, res.Body)
	}

	// The errors map must be keyed by the json field name, not the Go field name,
	// or a client cannot map it back to what it sent.
	if body := res.Body.String(); !strings.Contains(body, `"date"`) {
		t.Errorf("body = %s, want a field error keyed \"date\"", body)
	}
}

// Each event should land on a distinct day, which is what the factory sequence
// is for.
func TestEventFactorySpreadsDates(t *testing.T) {
	support.FreshDatabase(t)

	owner, err := factories.UserFactory().CreateOne()
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}

	events, err := factories.EventFactory().Count(4).State(factories.ForUser(owner)).Create()
	if err != nil {
		t.Fatalf("creating events: %v", err)
	}

	dates := make(map[string]struct{}, len(events))

	for _, event := range events {
		if _, duplicate := dates[event.Date]; duplicate {
			t.Errorf("date %s was reused; the sequence is not advancing", event.Date)
		}

		dates[event.Date] = struct{}{}
	}
}

func TestEventsStillRequiresAuthWithRealRoutes(t *testing.T) {
	support.FreshDatabase(t)

	router, _ := support.AppRouter(t)

	tests := []struct {
		name   string
		header string
	}{
		{name: "no header", header: ""},
		{name: "shorter than the Bearer prefix", header: "abc"},
		{name: "garbage token", header: middlewares.BearerPrefix + "not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := support.GetPath(t, router, support.V1Path("/events"), tt.header); res.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (body: %s)", res.Code, http.StatusUnauthorized, res.Body)
			}
		})
	}
}

// EventFactory leaves UserID unset on purpose, so a bare Create must fail on
// the foreign key rather than quietly attaching to whatever id exists.
func TestEventFactoryWithoutOwnerFails(t *testing.T) {
	support.FreshDatabase(t)

	if _, err := factories.EventFactory().CreateOne(); err == nil {
		t.Error("CreateOne() = nil error, want a foreign key failure")
	}
}

// Make must not touch the database.
func TestEventFactoryMakeDoesNotPersist(t *testing.T) {
	support.FreshDatabase(t)

	made, err := factories.EventFactory().Count(2).Make()
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	if len(made) != 2 {
		t.Fatalf("made %d events, want 2", len(made))
	}

	for _, event := range made {
		if event.ID != 0 {
			t.Errorf("made event has id %d, want 0 — Make must not insert", event.ID)
		}
	}

	stored, err := models.NewEventModel().Paginate(10, 1)
	if err != nil {
		t.Fatalf("Paginate() = %v, want nil", err)
	}

	if len(stored) != 0 {
		t.Errorf("found %d stored events, want 0", len(stored))
	}
}
