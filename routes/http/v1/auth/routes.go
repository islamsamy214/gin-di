package auth

import (
	"web-app/app/container"
	controllers "web-app/app/http/controllers/v1/auth"
	"web-app/app/http/middlewares"

	"github.com/gin-gonic/gin"
)

/*
 * Register mounts the public authentication routes.
 *
 * @param router The group these routes hang from.
 * @param c      The resolved application the controller is built from.
 */
func Register(router *gin.RouterGroup, c *container.Container) {
	controller := controllers.NewController(c.Auth(), c.Users())

	/*
	 * A tighter allowance than the global one, applied to this route alone.
	 *
	 * Login is the expensive endpoint: every attempt runs an argon2id
	 * verification that allocates 64 MiB, so unlimited retries are costly as
	 * well as a brute-force surface. The service caps how many run at once,
	 * which stops the process falling over but does nothing about how many an
	 * attacker may make.
	 *
	 * This composes with the global throttle rather than replacing it, because
	 * the two allowances are counted under separate names. Applying it to a
	 * whole group instead is the same call on that group's Use.
	 */
	router.POST("/login",
		middlewares.Throttle(c.Limiter(), c.Throttle().Login),
		controller.Login,
	)
}
