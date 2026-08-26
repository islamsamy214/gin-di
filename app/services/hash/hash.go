/*
 * Package hash derives and verifies password hashes, mirroring Laravel's Hash
 * facade.
 *
 * Its own package rather than free functions sitting beside the token service:
 * hashing and token issuance share no state and change for unrelated reasons,
 * and separating them lets the token service stay constructor-injected while
 * these stay pure functions.
 */
package hash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

/*
 * Argon2id parameters. Changing any of them invalidates every stored hash,
 * because the encoding carries the salt but not the cost factors.
 *
 * memoryKiB is the reason password verification has to be bounded elsewhere:
 * each concurrent call allocates this much, so unlimited concurrency on the
 * login route is a memory-exhaustion vector rather than merely slow.
 */
const (
	timeCost   = 1
	memoryKiB  = 64 * 1024
	threads    = 4
	keyLength  = 32
	saltLength = 16
)

// ErrInvalidHash reports a stored value that is not a hash this package wrote.
var ErrInvalidHash = errors.New("invalid hash format")

/*
 * Make derives a hash from a plaintext password.
 *
 * @param password The plaintext to hash.
 * @return string The base64-encoded salt and hash.
 * @return error  If the salt could not be generated.
 */
func Make(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	derived := argon2.IDKey([]byte(password), salt, timeCost, memoryKiB, threads, keyLength)

	// The salt is prepended rather than stored separately, so a single column
	// holds everything Check needs.
	return base64.StdEncoding.EncodeToString(append(salt, derived...)), nil
}

/*
 * Check reports whether a plaintext password matches a stored hash.
 *
 * The comparison is constant-time: a short-circuiting one leaks how much of the
 * derived key matched, which is enough to recover it byte by byte.
 *
 * @param hashed   The stored value, as produced by Make.
 * @param password The plaintext candidate.
 * @return bool  Whether they match.
 * @return error If the stored value is not a well-formed hash.
 */
func Check(hashed, password string) (bool, error) {
	data, err := base64.StdEncoding.DecodeString(hashed)
	if err != nil {
		return false, fmt.Errorf("decoding hash: %w", err)
	}

	if len(data) < saltLength {
		return false, ErrInvalidHash
	}

	salt, stored := data[:saltLength], data[saltLength:]
	derived := argon2.IDKey([]byte(password), salt, timeCost, memoryKiB, threads, keyLength)

	return subtle.ConstantTimeCompare(derived, stored) == 1, nil
}
