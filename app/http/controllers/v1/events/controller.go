package events

import (
	"net/http"
	"web-app/app/auth"
	"web-app/app/exceptions"
	"web-app/app/http/requests/events"
	"web-app/app/http/resources"
	"web-app/app/http/responses"
	"web-app/app/models"
	"web-app/app/services/core"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	db *core.PostgresService
}

// NewController builds the controller against the shared connection, rather than
// letting each request resolve its own.
func NewController(db *core.PostgresService) *Controller {
	return &Controller{db: db}
}

/*
 * Index lists the authenticated user's events.
 *
 * @route GET /api/v1/events
 */
func (controller *Controller) Index(ctx *gin.Context) {
	userID, ok := auth.ID(ctx)
	if !ok {
		// Unreachable behind Authenticate, but a route mounted on the wrong group
		// must fail closed rather than fall back to reading everyone's rows.
		_ = ctx.Error(exceptions.NewUnauthorized())

		return
	}

	var request events.IndexRequest

	if err := ctx.ShouldBindQuery(&request); err != nil {
		_ = ctx.Error(exceptions.FromBindError(err))

		return
	}

	request.Defaults()

	/*
	 * PaginateForUser, not Paginate. The unscoped call this replaced returned
	 * every user's events to any authenticated caller — the tests only looked
	 * correct because a fresh database held a single user.
	 */
	found, err := models.NewEventModel().PaginateForUser(userID, request.PerPage, request.Page)
	if err != nil {
		_ = ctx.Error(exceptions.FromDatabaseError(err))

		return
	}

	responses.Success(ctx, http.StatusOK, "", resources.NewEventPage(
		resources.NewEventCollection(found),
		request.Page,
		request.PerPage,
	))
}

/*
 * Create stores a new event owned by the authenticated user.
 *
 * @route POST /api/v1/events
 */
func (controller *Controller) Create(ctx *gin.Context) {
	userID, ok := auth.ID(ctx)
	if !ok {
		_ = ctx.Error(exceptions.NewUnauthorized())

		return
	}

	var request events.StoreRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		_ = ctx.Error(exceptions.FromBindError(err))

		return
	}

	event := models.NewEventModel()
	event.Name = request.Name
	event.Date = request.Date

	// Ownership comes from the verified token, never from the request body. This
	// used to be hardcoded to 1, so every event created through the API was
	// attributed to whoever happened to hold that id.
	event.UserID = userID

	if err := event.Create(); err != nil {
		_ = ctx.Error(exceptions.FromDatabaseError(err))

		return
	}

	responses.Success(ctx, http.StatusCreated, "Event created", resources.NewSingleEvent(event))
}
