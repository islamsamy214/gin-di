package events

import (
	"net/http"
	"web-app/app/models"

	"github.com/gin-gonic/gin"
)

type Controller struct{}

func NewController() *Controller {
	return &Controller{}
}

func (controller *Controller) Index(ctx *gin.Context) {
	eventsModel := models.NewEventModel()
	events, err := eventsModel.Paginate(10, 1)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": events})
}

func (controller *Controller) Create(ctx *gin.Context) {
	eventsModel := models.NewEventModel()
	if err := ctx.ShouldBindJSON(eventsModel); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	eventsModel.UserId = 1
	if err := eventsModel.Create(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"data": eventsModel})
}
