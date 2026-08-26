/*
 * Package auth holds the form requests for the authentication endpoints,
 * mirroring Laravel's app/Http/Requests.
 */
package auth

/*
 * LoginRequest is the bind target for POST /login.
 *
 * Separate from models.User deliberately. A model carries a stored password
 * hash and a request carries a plaintext candidate; letting one struct be both
 * the bind target and the response body is exactly how that hash ended up
 * serialized to clients on every successful login.
 *
 * The max on the password is a control rather than decoration: every submitted
 * value is run through argon2id, so an unbounded field lets a caller choose how
 * much work and memory the server spends rejecting them. The limits live only in
 * the tags because struct tags cannot reference constants, and two declarations
 * of the same bound would drift.
 */
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=1024"`
}
