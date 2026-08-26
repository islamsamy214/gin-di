package unit

import (
	"errors"
	"testing"
	"web-app/app/services/hash"
)

// Moved here from auth_service_test.go when password hashing was split out of
// AuthService into the Hash facade. The assertions are unchanged.

func TestMakeAndCheck(t *testing.T) {
	const password = "correct-horse-battery-staple"

	hashed, err := hash.Make(password)
	if err != nil {
		t.Fatalf("Make() = %v, want nil", err)
	}

	t.Run("accepts the correct password", func(t *testing.T) {
		match, err := hash.Check(hashed, password)
		if err != nil {
			t.Fatalf("Check() = %v, want nil", err)
		}

		if !match {
			t.Error("Check() = false, want true")
		}
	})

	t.Run("rejects the wrong password", func(t *testing.T) {
		match, err := hash.Check(hashed, "wrong-password")
		if err != nil {
			t.Fatalf("Check() = %v, want nil", err)
		}

		if match {
			t.Error("Check() = true, want false")
		}
	})

	t.Run("salt is random per hash", func(t *testing.T) {
		other, err := hash.Make(password)
		if err != nil {
			t.Fatalf("Make() = %v, want nil", err)
		}

		if other == hashed {
			t.Error("two hashes of the same password are identical, want distinct salts")
		}
	})
}

func TestCheckRejectsMalformedHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{name: "not base64", hash: "!!!not-base64!!!"},
		{name: "shorter than the salt", hash: "AAAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := hash.Check(tt.hash, "any-password")
			if err == nil {
				t.Error("Check() = nil error, want an error")
			}

			if match {
				t.Error("Check() = true, want false")
			}
		})
	}
}

// A stored value that is valid base64 but too short to contain a salt is a
// distinct failure from unparseable input, and callers switch on it.
func TestCheckReportsInvalidHashSentinel(t *testing.T) {
	if _, err := hash.Check("AAAA", "any-password"); !errors.Is(err, hash.ErrInvalidHash) {
		t.Errorf("Check() = %v, want %v", err, hash.ErrInvalidHash)
	}
}
