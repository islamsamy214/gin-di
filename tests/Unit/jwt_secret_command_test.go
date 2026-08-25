package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"web-app/app/console"
	"web-app/configs"
)

const envFixture = `APP_NAME=WebApplication
APP_PORT=8000

JWT_SECRET=
JWT_ISSUER=github@islamsamy214
JWT_TTL=86400

DB_HOST=127.0.0.1
`

// envDir puts the command in a throwaway directory holding the given .env, so
// no test can touch the repository's real one.
func envDir(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing fixture .env: %v", err)
	}

	t.Chdir(dir)

	return path
}

func readEnv(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}

	return string(contents)
}

// secretIn returns the JWT_SECRET value assigned in the given .env contents.
func secretIn(t *testing.T, contents string) string {
	t.Helper()

	for _, line := range strings.Split(contents, "\n") {
		if value, ok := strings.CutPrefix(line, "JWT_SECRET="); ok {
			return value
		}
	}

	t.Fatal("no JWT_SECRET line found")

	return ""
}

func TestGenerateJwtSecretMeetsConfigMinimum(t *testing.T) {
	secret, err := console.GenerateJwtSecret()
	if err != nil {
		t.Fatalf("GenerateJwtSecret() = %v, want nil", err)
	}

	if len(secret) < configs.MinSecretKeyLength {
		t.Errorf("secret is %d characters, want at least %d", len(secret), configs.MinSecretKeyLength)
	}
}

// The generated secret must satisfy the very config that will consume it.
func TestGeneratedSecretIsAcceptedByConfig(t *testing.T) {
	secret, err := console.GenerateJwtSecret()
	if err != nil {
		t.Fatalf("GenerateJwtSecret() = %v, want nil", err)
	}

	t.Setenv("JWT_SECRET", secret)

	if _, err := configs.NewJwtConfig(); err != nil {
		t.Errorf("NewJwtConfig() = %v, want nil", err)
	}
}

// A repeated secret would mean the randomness is broken.
func TestGenerateJwtSecretIsUnique(t *testing.T) {
	const runs = 100

	seen := make(map[string]struct{}, runs)

	for range runs {
		secret, err := console.GenerateJwtSecret()
		if err != nil {
			t.Fatalf("GenerateJwtSecret() = %v, want nil", err)
		}

		if _, duplicate := seen[secret]; duplicate {
			t.Fatalf("GenerateJwtSecret() returned a duplicate after %d runs", len(seen))
		}

		seen[secret] = struct{}{}
	}
}

// URL-safe base64 keeps the value free of characters that complicate .env.
func TestGenerateJwtSecretIsEnvSafe(t *testing.T) {
	secret, err := console.GenerateJwtSecret()
	if err != nil {
		t.Fatalf("GenerateJwtSecret() = %v, want nil", err)
	}

	if strings.ContainsAny(secret, "+/=\"'$ \t\n#") {
		t.Errorf("secret %q contains characters unsafe for .env", secret)
	}
}

func TestJwtSecretCommandFillsEmptySecret(t *testing.T) {
	path := envDir(t, envFixture)

	if err := console.NewJwtSecretCommand().Handle(nil); err != nil {
		t.Fatalf("Handle() = %v, want nil", err)
	}

	contents := readEnv(t, path)

	if secret := secretIn(t, contents); len(secret) < configs.MinSecretKeyLength {
		t.Errorf("JWT_SECRET = %q, want at least %d characters", secret, configs.MinSecretKeyLength)
	}

	// Every other setting must survive untouched.
	for _, line := range []string{"APP_NAME=WebApplication", "APP_PORT=8000", "JWT_ISSUER=github@islamsamy214", "JWT_TTL=86400", "DB_HOST=127.0.0.1"} {
		if !strings.Contains(contents, line) {
			t.Errorf(".env lost %q\n--- got ---\n%s", line, contents)
		}
	}
}

// Replacing a live secret invalidates every issued token, so it needs --force.
func TestJwtSecretCommandRefusesToOverwriteWithoutForce(t *testing.T) {
	existing := strings.Replace(envFixture, "JWT_SECRET=", "JWT_SECRET=already-set-and-long-enough-to-be-real", 1)
	path := envDir(t, existing)

	if err := console.NewJwtSecretCommand().Handle(nil); err == nil {
		t.Error("Handle() = nil error, want a refusal")
	}

	if got := readEnv(t, path); got != existing {
		t.Errorf(".env was modified despite the refusal\n--- got ---\n%s", got)
	}
}

func TestJwtSecretCommandForceReplacesExisting(t *testing.T) {
	const old = "already-set-and-long-enough-to-be-real"

	path := envDir(t, strings.Replace(envFixture, "JWT_SECRET=", "JWT_SECRET="+old, 1))

	if err := console.NewJwtSecretCommand().Handle([]string{"--force"}); err != nil {
		t.Fatalf("Handle() = %v, want nil", err)
	}

	secret := secretIn(t, readEnv(t, path))

	if secret == old {
		t.Error("JWT_SECRET was not replaced")
	}

	if len(secret) < configs.MinSecretKeyLength {
		t.Errorf("JWT_SECRET = %q, want at least %d characters", secret, configs.MinSecretKeyLength)
	}
}

// A .env without the key at all should gain it rather than error.
func TestJwtSecretCommandAppendsWhenKeyAbsent(t *testing.T) {
	path := envDir(t, "APP_NAME=WebApplication\n")

	if err := console.NewJwtSecretCommand().Handle(nil); err != nil {
		t.Fatalf("Handle() = %v, want nil", err)
	}

	contents := readEnv(t, path)

	if !strings.Contains(contents, "APP_NAME=WebApplication") {
		t.Errorf(".env lost APP_NAME\n--- got ---\n%s", contents)
	}

	if secret := secretIn(t, contents); len(secret) < configs.MinSecretKeyLength {
		t.Errorf("JWT_SECRET = %q, want at least %d characters", secret, configs.MinSecretKeyLength)
	}
}

// --show must not touch the file, so it is safe to pipe anywhere.
func TestJwtSecretCommandShowLeavesEnvAlone(t *testing.T) {
	path := envDir(t, envFixture)

	if err := console.NewJwtSecretCommand().Handle([]string{"--show"}); err != nil {
		t.Fatalf("Handle() = %v, want nil", err)
	}

	if got := readEnv(t, path); got != envFixture {
		t.Errorf(".env was modified by --show\n--- got ---\n%s", got)
	}
}

func TestJwtSecretCommandFailsWithoutEnvFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := console.NewJwtSecretCommand().Handle(nil); err == nil {
		t.Error("Handle() = nil error, want an error naming the missing .env")
	}
}

func TestJwtSecretCommandPreservesFileMode(t *testing.T) {
	path := envDir(t, envFixture)

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := console.NewJwtSecretCommand().Handle(nil); err != nil {
		t.Fatalf("Handle() = %v, want nil", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode = %o, want 600 — a secrets file must not widen", mode)
	}
}

func TestJwtSecretCommandRejectsUnknownFlag(t *testing.T) {
	envDir(t, envFixture)

	if err := console.NewJwtSecretCommand().Handle([]string{"--nope"}); err == nil {
		t.Error("Handle() = nil error, want an error")
	}
}

func TestJwtSecretCommandHasDescription(t *testing.T) {
	if console.NewJwtSecretCommand().Description() == "" {
		t.Error("Description() = empty, want a description")
	}
}
