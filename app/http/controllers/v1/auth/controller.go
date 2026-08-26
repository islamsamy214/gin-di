package auth

import (
	"net/http"
	"web-app/app/exceptions"
	requests "web-app/app/http/requests/auth"
	"web-app/app/http/resources"
	"web-app/app/http/responses"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	auth  *services.AuthService
	users *services.UserService
}

func NewController(auth *services.AuthService, users *services.UserService) *Controller {
	return &Controller{auth: auth, users: users}
}

/*
 * Login exchanges credentials for an access token.
 *
 * @route POST /api/v1/login
 */
func (controller *Controller) Login(ctx *gin.Context) {
	var request requests.LoginRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		_ = ctx.Error(exceptions.FromBindError(err))

		return
	}

	user, err := controller.users.Authenticate(request.Username, request.Password)
	if err != nil {
		/*
		 * One response for "no such user" and "wrong password", and a 401 rather
		 * than the 400 this used to return. Distinguishing the two cases turns
		 * the endpoint into a username-enumeration oracle, and the cause is
		 * carried on the exception so the log still records which it was.
		 */
		unauthorized := exceptions.NewUnauthorized()
		unauthorized.Message = "Invalid credentials"
		unauthorized.Err = err

		_ = ctx.Error(unauthorized)

		return
	}

	token, err := controller.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		_ = ctx.Error(exceptions.NewInternal(err))

		return
	}

	/*
	 * The user goes out as a resource, never as the model. models.User carries
	 * the stored argon2id hash, and returning the model here served that hash to
	 * the client on every successful login.
	 */
	responses.Success(ctx, http.StatusOK, "Logged in", resources.NewSessionResource(token, user))
}
