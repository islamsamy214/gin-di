package controllers

import (
	"net/http"
	"web-app/app/models"

	"github.com/gin-gonic/gin"
)

type EventController struct{}

func NewEventController() *EventController {
	return &EventController{}
}

func (e *EventController) Index(c *gin.Context) {
	eventsModel := models.NewEventModel()
	events, err := eventsModel.Paginate(10, 1)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events})
}

func (e *EventController) Create(c *gin.Context) {
	eventsModel := models.NewEventModel()
	if err := c.ShouldBindJSON(eventsModel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	eventsModel.UserId = 1
	if err := eventsModel.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": eventsModel})
}
