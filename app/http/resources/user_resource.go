/*
 * Package resources shapes models for the wire, mirroring Laravel's
 * app/Http/Resources.
 *
 * A model is never handed to a response helper directly. The resource is an
 * allowlist, so a column added to a table cannot reach a client by default —
 * which is the failure mode that leaked stored password hashes on every login.
 */
package resources

import "web-app/app/models"

/*
 * UserResource is the public projection of a user.
 *
 * There is no password field and there must never be one. models.User also tags
 * its hash json:"-", so the leak is blocked twice over: once because this type
 * cannot express it, and once because even marshalling the model directly would
 * omit it.
 */
type UserResource struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// NewUserResource projects a single user.
func NewUserResource(user *models.User) UserResource {
	return UserResource{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}
}

/*
 * SessionResource is the payload of a successful login.
 *
 * A named type rather than an inline map, for the same reason as
 * EventCollection: the wire shape is declared in one place, and a handler cannot
 * accidentally ship a different set of keys.
 */
type SessionResource struct {
	Token string       `json:"token"`
	User  UserResource `json:"user"`
}

// NewSessionResource pairs an issued token with the user it was issued to.
func NewSessionResource(token string, user *models.User) SessionResource {
	return SessionResource{Token: token, User: NewUserResource(user)}
}
