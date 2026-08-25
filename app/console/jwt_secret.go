package console

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"web-app/configs"
)

const (
	envFileName  = ".env"
	jwtSecretKey = "JWT_SECRET"

	// 32 bytes of entropy: the HMAC-SHA256 key size. Base64 widens this to 43
	// characters, comfortably past configs.MinSecretKeyLength.
	secretRandomBytes = 32
)

type JwtSecretCommand struct{}

func NewJwtSecretCommand() *JwtSecretCommand {
	return &JwtSecretCommand{}
}

/*
 * Handle generates a signing secret and stores it in .env.
 *
 * A fresh clone has no JWT_SECRET and the app refuses to boot without one, so
 * this is the first command a newcomer runs.
 */
func (command *JwtSecretCommand) Handle(args []string) error {
	flags := flag.NewFlagSet("jwt:secret", flag.ContinueOnError)
	show := flags.Bool("show", false, "print the secret instead of writing it to .env")
	force := flags.Bool("force", false, "replace an existing JWT_SECRET, invalidating every issued token")

	if err := flags.Parse(args); err != nil {
		// -h already printed usage; that is not a failure.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parsing flags: %w", err)
	}

	secret, err := GenerateJwtSecret()
	if err != nil {
		return err
	}

	if *show {
		fmt.Println(secret)
		return nil
	}

	if err := writeJwtSecret(envFileName, secret, *force); err != nil {
		return err
	}

	log.Printf("%s set in %s", jwtSecretKey, envFileName)

	return nil
}

func (command *JwtSecretCommand) Description() string {
	return "Generates a signing secret and writes it to .env (--show, --force)"
}

/*
 * GenerateJwtSecret returns a fresh base64 signing secret.
 *
 * Uses crypto/rand: a predictable secret lets anyone mint valid tokens. The
 * URL-safe alphabet avoids the "+/=" characters that complicate .env values.
 */
func GenerateJwtSecret() (string, error) {
	buf := make([]byte, secretRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	secret := base64.RawURLEncoding.EncodeToString(buf)

	// Guard the encoding against ever dropping below what the config accepts.
	if len(secret) < configs.MinSecretKeyLength {
		return "", fmt.Errorf("generated secret is %d characters, want at least %d", len(secret), configs.MinSecretKeyLength)
	}

	return secret, nil
}

// writeJwtSecret replaces the JWT_SECRET line in path, appending it if absent.
// Every other line is preserved byte for byte.
func writeJwtSecret(path, secret string, force bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found: copy .env.example to %s first", path, path)
		}

		return fmt.Errorf("reading %s: %w", path, err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(contents), "\n")
	assignment := jwtSecretKey + "=" + secret
	replaced := false

	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), jwtSecretKey+"=") {
			continue
		}

		existing := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), jwtSecretKey+"="))
		if existing != "" && !force {
			return fmt.Errorf("%s already has a value in %s; pass --force to replace it, which invalidates every issued token", jwtSecretKey, path)
		}

		lines[i] = assignment
		replaced = true

		break
	}

	if !replaced {
		lines = append(lines, assignment)
	}

	return replaceFile(path, strings.Join(lines, "\n"), info.Mode())
}

// replaceFile swaps path's contents via a sibling temp file, so a failure part
// way through cannot leave a half-written .env behind.
func replaceFile(path, contents string, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(contents); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return fmt.Errorf("setting mode on temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}

	return nil
}
