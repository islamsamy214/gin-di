/*
 * Package auth reads the authenticated identity off a request, mirroring
 * Laravel's Auth facade.
 *
 * It owns the context keys, so the middleware that writes them and the handlers
 * that read them cannot drift apart by mistyping a string literal. A leaf
 * package with no application imports, so nothing can cycle through it.
 */
package auth

import "github.com/gin-gonic/gin"

/*
 * The keys the identity is stored under.
 *
 * The literal values are load-bearing: existing handlers and tests read them
 * with ctx.GetInt64("userId") / ctx.GetString("username"), so they are kept as
 * they were rather than renamed alongside this refactor.
 */
const (
	contextUserID   = "userId"
	contextUsername = "username"
)

/*
 * SetUser records the identity the authentication middleware resolved.
 *
 * @param ctx      The request being authenticated.
 * @param userID   The authenticated user's id.
 * @param username The authenticated user's name.
 */
func SetUser(ctx *gin.Context, userID int64, username string) {
	ctx.Set(contextUserID, userID)
	ctx.Set(contextUsername, username)
}

/*
 * ID returns the authenticated user's id.
 *
 * The boolean is false when the request never passed the authentication
 * middleware, so a handler cannot mistake "nobody is logged in" for "user 0".
 * That distinction matters: the events controller previously hardcoded an owner
 * id rather than reading one, and a silent zero would have been the same class
 * of bug wearing a different number.
 *
 * @param ctx The request to read.
 * @return int64 The user id, zero when absent.
 * @return bool  Whether an authenticated identity was present.
 */
func ID(ctx *gin.Context) (int64, bool) {
	value, exists := ctx.Get(contextUserID)
	if !exists {
		return 0, false
	}

	id, ok := value.(int64)

	return id, ok && id != 0
}

/*
 * Username returns the authenticated user's name, empty when absent.
 *
 * @param ctx The request to read.
 */
func Username(ctx *gin.Context) string {
	return ctx.GetString(contextUsername)
}
