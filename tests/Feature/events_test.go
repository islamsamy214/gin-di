package feature

import (
	"encoding/json"
	"net/http"
	"testing"
	"web-app/app/http/middlewares"
	"web-app/app/models"
	"web-app/database/factories"
)

// eventsFrom decodes the {"data": [...]} envelope the events endpoint returns.
func eventsFrom(t *testing.T, body []byte) []models.Event {
	t.Helper()

	var parsed struct {
		Data []models.Event `json:"data"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decoding events response %s: %v", body, err)
	}

	return parsed.Data
}

func TestEventsReturnsFactoryEvents(t *testing.T) {
	freshDatabase(t)

	router, auth := appRouter(t)

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

	res := getPath(t, router, v1Path("/events"), middlewares.BearerPrefix+token)
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
		if event.UserId != owner.ID {
			t.Errorf("event %q has user_id %d, want %d", event.Name, event.UserId, owner.ID)
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

// Each event should land on a distinct day, which is what the factory sequence
// is for.
func TestEventFactorySpreadsDates(t *testing.T) {
	freshDatabase(t)

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
	freshDatabase(t)

	router, _ := appRouter(t)

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
			if res := getPath(t, router, v1Path("/events"), tt.header); res.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (body: %s)", res.Code, http.StatusUnauthorized, res.Body)
			}
		})
	}
}
