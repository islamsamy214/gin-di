package services

import (
	"errors"
	"fmt"
	"web-app/app/models"
	"web-app/app/services/core"
	"web-app/app/services/hash"
)

/*
 * maxConcurrentVerifications bounds how many password checks run at once.
 *
 * Each argon2id verification allocates 64 MiB, so an unauthenticated caller
 * could previously turn a few hundred concurrent login attempts into gigabytes
 * of resident memory. Four keeps the ceiling at 256 MiB per process; excess
 * requests queue rather than allocate, which is the correct failure mode for an
 * endpoint whose cost is deliberately high.
 */
const maxConcurrentVerifications = 4

// ErrInvalidCredentials is returned for both an unknown username and a wrong
// password. One sentinel for both, because distinguishing them tells an attacker
// which usernames exist.
var ErrInvalidCredentials = errors.New("invalid credentials")

/*
 * UserService owns authentication of stored credentials.
 *
 * Split out of AuthService, which now does tokens only: verifying a password
 * needs a database and a hasher, issuing a token needs a signing key, and one
 * type holding all three could not be constructed in a test without a database.
 */
type UserService struct {
	db *core.PostgresService

	// A buffered channel as a counting semaphore. Nothing is ever read from the
	// values, only slots taken and released.
	verifications chan struct{}
}

// NewUserService builds the service against an injected connection.
func NewUserService(db *core.PostgresService) *UserService {
	return &UserService{
		db:            db,
		verifications: make(chan struct{}, maxConcurrentVerifications),
	}
}

/*
 * Authenticate verifies a username and password against the stored hash.
 *
 * @param username The claimed identity.
 * @param password The plaintext candidate.
 * @return *models.User The matching user on success.
 * @return error        ErrInvalidCredentials when the credentials do not match,
 *                      or a wrapped cause when the lookup itself failed.
 */
func (service *UserService) Authenticate(username, password string) (*models.User, error) {
	user := models.NewUserModel()
	user.Username = username

	if err := user.FindByUsername(); err != nil {
		// A missing row and a broken database are different operational events,
		// but the caller learns the same thing either way: these credentials do
		// not work. The cause is preserved for the log.
		return nil, fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}

	match, err := service.verify(user.Password, password)
	if err != nil {
		return nil, fmt.Errorf("verifying password: %w", err)
	}

	if !match {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

/*
 * verify runs one password check, waiting for a free slot first.
 *
 * @param hashed   The stored hash.
 * @param password The plaintext candidate.
 */
func (service *UserService) verify(hashed, password string) (bool, error) {
	service.verifications <- struct{}{}
	defer func() { <-service.verifications }()

	return hash.Check(hashed, password)
}
