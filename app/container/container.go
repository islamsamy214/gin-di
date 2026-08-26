/*
 * Package container holds the resolved application, mirroring Laravel's service
 * container.
 *
 * It exists so route registration can thread dependencies downward without every
 * Register signature growing a parameter per collaborator. Domain Register
 * functions pull the typed dependencies their controller needs out of it —
 * controllers themselves never receive the container, so a controller's
 * constructor still states exactly what it depends on.
 *
 * There is no reflection, no lifetime management and no lookup by name: this is
 * a struct of already-constructed collaborators, resolved once at boot. A
 * missing dependency is a compile error rather than a runtime one.
 */
package container

import (
	"log/slog"
	"web-app/app/services"
	"web-app/app/services/core"
	"web-app/configs"
)

// Config carries everything the container is built from. A separate type from
// Container so callers construct it with field names, which keeps the call site
// readable as the set of dependencies grows.
type Config struct {
	App    *configs.AppConfig
	Auth   *services.AuthService
	Users  *services.UserService
	DB     *core.PostgresService
	Logger *slog.Logger
}

// Container exposes the resolved application to route registration.
type Container struct {
	app    *configs.AppConfig
	auth   *services.AuthService
	users  *services.UserService
	db     *core.PostgresService
	logger *slog.Logger
}

// New resolves a container from its dependencies.
func New(config Config) *Container {
	return &Container{
		app:    config.App,
		auth:   config.Auth,
		users:  config.Users,
		db:     config.DB,
		logger: config.Logger,
	}
}

// App returns the application configuration.
func (container *Container) App() *configs.AppConfig { return container.app }

// Auth returns the token service.
func (container *Container) Auth() *services.AuthService { return container.auth }

// Users returns the user service.
func (container *Container) Users() *services.UserService { return container.users }

// DB returns the shared database connection.
func (container *Container) DB() *core.PostgresService { return container.db }

// Logger returns the application logger.
func (container *Container) Logger() *slog.Logger { return container.logger }
