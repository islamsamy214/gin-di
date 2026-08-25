package v1

import (
	"net/http"
	"web-app/app/models"
	"web-app/app/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (controller *AuthController) Login(ctx *gin.Context) {
	user := models.NewUserModel()
	err := ctx.ShouldBindJSON(&user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loginUser, err := controller.auth.AttemptLogin(user, user.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := controller.auth.GenerateToken(loginUser.ID, loginUser.Username)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Success",
		"user":    loginUser,
		"token":   token,
	})
}
