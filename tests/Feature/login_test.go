package feature

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"web-app/app/models"
	"web-app/database/factories"
)

// loginBody builds a credentials payload for POST /login.
func loginBody(username, password string) string {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		panic(fmt.Sprintf("marshalling login body: %v", err))
	}

	return string(body)
}

// tokenFrom pulls the token out of a successful login response.
func tokenFrom(t *testing.T, body []byte) string {
	t.Helper()

	var parsed struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decoding login response %s: %v", body, err)
	}

	if parsed.Token == "" {
		t.Fatalf("login response carried no token: %s", body)
	}

	return parsed.Token
}

func TestLoginWithFactoryUser(t *testing.T) {
	freshDatabase(t)

	router, _ := appRouter(t)

	const username = "alice"

	if _, err := factories.UserFactory().State(factories.WithUsername(username)).CreateOne(); err != nil {
		t.Fatalf("creating the factory user: %v", err)
	}

	res := postJSON(t, router, v1Path("/login"), loginBody(username, factories.FactoryPassword))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	// The token must be usable, not merely present.
	token := tokenFrom(t, res.Body.Bytes())

	if _, err := authService(t, testSecret).ParseToken(token); err != nil {
		t.Errorf("ParseToken() = %v, want the issued token to verify", err)
	}
}

// The token must carry the row's real id, not a placeholder.
func TestLoginTokenCarriesTheUsersID(t *testing.T) {
	freshDatabase(t)

	router, auth := appRouter(t)

	const username = "bob"

	user, err := factories.UserFactory().State(factories.WithUsername(username)).CreateOne()
	if err != nil {
		t.Fatalf("creating the factory user: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("factory user has id 0, so Create did not read back the inserted row")
	}

	res := postJSON(t, router, v1Path("/login"), loginBody(username, factories.FactoryPassword))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	claims, err := auth.ParseToken(tokenFrom(t, res.Body.Bytes()))
	if err != nil {
		t.Fatalf("ParseToken() = %v, want nil", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("token UserID = %d, want %d", claims.UserID, user.ID)
	}

	if claims.Username != username {
		t.Errorf("token Username = %q, want %q", claims.Username, username)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	freshDatabase(t)

	router, _ := appRouter(t)

	const username = "carol"

	if _, err := factories.UserFactory().State(factories.WithUsername(username)).CreateOne(); err != nil {
		t.Fatalf("creating the factory user: %v", err)
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "wrong password", body: loginBody(username, "not-the-password")},
		{name: "unknown user", body: loginBody("nobody", factories.FactoryPassword)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := postJSON(t, router, v1Path("/login"), tt.body)

			if res.Code == http.StatusOK {
				t.Errorf("status = %d, want a rejection (body: %s)", res.Code, res.Body)
			}
		})
	}
}

// Every factory user shares one password, so a batch must all be able to log in.
func TestFactoryBatchUsersCanAllLogIn(t *testing.T) {
	freshDatabase(t)

	router, _ := appRouter(t)

	users, err := factories.UserFactory().Count(3).Create()
	if err != nil {
		t.Fatalf("creating factory users: %v", err)
	}

	if len(users) != 3 {
		t.Fatalf("created %d users, want 3", len(users))
	}

	seen := make(map[string]struct{}, len(users))

	for _, user := range users {
		if _, duplicate := seen[user.Username]; duplicate {
			t.Errorf("username %q was generated twice; the sequence suffix is not working", user.Username)
		}

		seen[user.Username] = struct{}{}

		res := postJSON(t, router, v1Path("/login"), loginBody(user.Username, factories.FactoryPassword))
		if res.Code != http.StatusOK {
			t.Errorf("login as %q: status = %d, want %d (body: %s)", user.Username, res.Code, http.StatusOK, res.Body)
		}
	}
}

// EventFactory leaves UserId unset on purpose, so a bare Create must fail on
// the foreign key rather than quietly attaching to whatever id exists.
func TestEventFactoryWithoutOwnerFails(t *testing.T) {
	freshDatabase(t)

	if _, err := factories.EventFactory().CreateOne(); err == nil {
		t.Error("CreateOne() = nil error, want a foreign key failure")
	}
}

// Make must not touch the database.
func TestEventFactoryMakeDoesNotPersist(t *testing.T) {
	freshDatabase(t)

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
