package unit

import (
	"errors"
	"strings"
	"testing"
	"web-app/app/models"
	"web-app/app/services"
	"web-app/database/factories"
)

// Make needs no database, so the factory mechanics are testable in isolation.

func TestFactoryMakeDefaultsToOne(t *testing.T) {
	made, err := factories.UserFactory().Make()
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	if len(made) != 1 {
		t.Errorf("made %d users, want 1", len(made))
	}
}

func TestFactoryCountControlsBatchSize(t *testing.T) {
	for _, count := range []int{0, 1, 5} {
		made, err := factories.UserFactory().Count(count).Make()
		if err != nil {
			t.Fatalf("Count(%d).Make() = %v, want nil", count, err)
		}

		if len(made) != count {
			t.Errorf("Count(%d) made %d users, want %d", count, len(made), count)
		}
	}
}

func TestFactoryRejectsNegativeCount(t *testing.T) {
	if _, err := factories.UserFactory().Count(-1).Make(); err == nil {
		t.Error("Count(-1).Make() = nil error, want an error")
	}
}

// The sequence is what keeps a batch from colliding on unique-ish columns.
func TestFactorySequenceMakesDistinctUsernames(t *testing.T) {
	made, err := factories.UserFactory().Count(20).Make()
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	seen := make(map[string]struct{}, len(made))

	for _, user := range made {
		if _, duplicate := seen[user.Username]; duplicate {
			t.Errorf("username %q generated twice", user.Username)
		}

		seen[user.Username] = struct{}{}
	}
}

func TestFactoryStateOverridesTheDefinition(t *testing.T) {
	const username = "pinned"

	user, err := factories.UserFactory().State(factories.WithUsername(username)).MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	if user.Username != username {
		t.Errorf("Username = %q, want %q", user.Username, username)
	}
}

// States apply in order, so the last one wins.
func TestFactoryStatesApplyInOrder(t *testing.T) {
	user, err := factories.UserFactory().
		State(factories.WithUsername("first")).
		State(factories.WithUsername("second")).
		MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	if user.Username != "second" {
		t.Errorf("Username = %q, want the later state to win with %q", user.Username, "second")
	}
}

func TestFactoryStateAppliesToEveryInstance(t *testing.T) {
	const suffix = "-tagged"

	made, err := factories.UserFactory().
		Count(4).
		State(func(user *models.User) { user.Username += suffix }).
		Make()
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	for _, user := range made {
		if !strings.HasSuffix(user.Username, suffix) {
			t.Errorf("Username = %q, want it to end with %q", user.Username, suffix)
		}
	}
}

func TestMakeOneRequiresACountOfOne(t *testing.T) {
	if _, err := factories.UserFactory().Count(3).MakeOne(); err == nil {
		t.Error("Count(3).MakeOne() = nil error, want an error")
	}
}

// The whole point of the shared password: a factory user can be logged in.
func TestFactoryUserPasswordVerifies(t *testing.T) {
	user, err := factories.UserFactory().MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	matches, err := services.VerifyPassword(user.Password, factories.FactoryPassword)
	if err != nil {
		t.Fatalf("VerifyPassword() = %v, want nil", err)
	}

	if !matches {
		t.Error("VerifyPassword() = false, want the factory password to verify")
	}
}

func TestFactoryUserPasswordIsHashedNotPlaintext(t *testing.T) {
	user, err := factories.UserFactory().MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	if user.Password == factories.FactoryPassword {
		t.Error("Password is stored in plaintext")
	}
}

// Every user shares one hash, which is what keeps a large batch cheap.
func TestFactoryReusesTheCachedHash(t *testing.T) {
	made, err := factories.UserFactory().Count(3).Make()
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	for _, user := range made[1:] {
		if user.Password != made[0].Password {
			t.Error("factory users have different password hashes, so the cache is not being used")
		}
	}
}

func TestEventFactoryLeavesOwnerUnset(t *testing.T) {
	event, err := factories.EventFactory().MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	if event.UserId != 0 {
		t.Errorf("UserId = %d, want 0 so the caller must state an owner", event.UserId)
	}

	if event.Name == "" {
		t.Error("Name is empty, want a generated value")
	}

	if event.Date == "" {
		t.Error("Date is empty, want a generated value")
	}
}

func TestForUserStatesTheOwner(t *testing.T) {
	owner := models.NewUserModel()
	owner.ID = 77

	event, err := factories.EventFactory().State(factories.ForUser(owner)).MakeOne()
	if err != nil {
		t.Fatalf("MakeOne() = %v, want nil", err)
	}

	if event.UserId != owner.ID {
		t.Errorf("UserId = %d, want %d", event.UserId, owner.ID)
	}
}

// Definition failures must surface rather than yielding a half-built model.
func TestFactoryPropagatesDefinitionErrors(t *testing.T) {
	wanted := errors.New("definition failed")

	factory := factories.New(func(sequence int) (*models.User, error) {
		return nil, wanted
	})

	if _, err := factory.Make(); !errors.Is(err, wanted) {
		t.Errorf("Make() = %v, want it to wrap %v", err, wanted)
	}
}
