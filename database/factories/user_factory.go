package factories

import (
	"fmt"
	"strings"
	"sync"
	"web-app/app/models"
	"web-app/app/services/hash"

	"github.com/brianvoe/gofakeit/v7"
)

// FactoryPassword is the plaintext shared by every factory-built user, so tests
// and the seeded account can always log in with a known credential.
const FactoryPassword = "password"

/*
 * factoryPasswordHash caches the hash for the process.
 *
 * Argon2id is deliberately expensive (64MB, 4 threads), so hashing the same
 * literal once per generated user would dominate a seed or test run. Mirrors
 * the static $password cache in Laravel's UserFactory.
 */
var factoryPasswordHash = sync.OnceValues(func() (string, error) {
	return hash.Make(FactoryPassword)
})

/*
 * UserFactory builds users whose password is always FactoryPassword.
 *
 * @return *Factory[*models.User] A fluent factory.
 */
func UserFactory() *Factory[*models.User] {
	return New(func(sequence int) (*models.User, error) {
		hashed, err := factoryPasswordHash()
		if err != nil {
			return nil, fmt.Errorf("hashing the factory password: %w", err)
		}

		user := models.NewUserModel()

		// Sequence-suffixed so a batch cannot collide on username.
		user.Username = fmt.Sprintf("%s%d", strings.ToLower(gofakeit.FirstName()), sequence)
		user.Password = hashed

		return user, nil
	})
}

/*
 * WithUsername pins the username, for fixtures that must be logged into by name.
 *
 * @return func(*models.User) A state for Factory.State.
 */
func WithUsername(username string) func(*models.User) {
	return func(user *models.User) {
		user.Username = username
	}
}
