package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"web-app/database/factories"
	"web-app/tests/support"
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
	support.FreshDatabase(t)

	router, _ := support.AppRouter(t)

	const username = "alice"

	if _, err := factories.UserFactory().State(factories.WithUsername(username)).CreateOne(); err != nil {
		t.Fatalf("creating the factory user: %v", err)
	}

	res := support.PostJSON(t, router, support.V1Path("/login"), loginBody(username, factories.FactoryPassword))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	// The token must be usable, not merely present.
	token := tokenFrom(t, res.Body.Bytes())

	if _, err := support.AuthService(t, support.TestSecret).ParseToken(token); err != nil {
		t.Errorf("ParseToken() = %v, want the issued token to verify", err)
	}
}

// The token must carry the row's real id, not a placeholder.
func TestLoginTokenCarriesTheUsersID(t *testing.T) {
	support.FreshDatabase(t)

	router, auth := support.AppRouter(t)

	const username = "bob"

	user, err := factories.UserFactory().State(factories.WithUsername(username)).CreateOne()
	if err != nil {
		t.Fatalf("creating the factory user: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("factory user has id 0, so Create did not read back the inserted row")
	}

	res := support.PostJSON(t, router, support.V1Path("/login"), loginBody(username, factories.FactoryPassword))
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
	support.FreshDatabase(t)

	router, _ := support.AppRouter(t)

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
			res := support.PostJSON(t, router, support.V1Path("/login"), tt.body)

			if res.Code == http.StatusOK {
				t.Errorf("status = %d, want a rejection (body: %s)", res.Code, res.Body)
			}
		})
	}
}

// Every factory user shares one password, so a batch must all be able to log in.
func TestFactoryBatchUsersCanAllLogIn(t *testing.T) {
	support.FreshDatabase(t)

	router, _ := support.AppRouter(t)

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

		res := support.PostJSON(t, router, support.V1Path("/login"), loginBody(user.Username, factories.FactoryPassword))
		if res.Code != http.StatusOK {
			t.Errorf("login as %q: status = %d, want %d (body: %s)", user.Username, res.Code, http.StatusOK, res.Body)
		}
	}
}
