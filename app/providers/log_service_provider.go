package providers

import (
	"web-app/app/services/core"

	"github.com/gin-gonic/gin"
)

type LogServiceProvider struct{}

func NewLogServiceProvider() *LogServiceProvider {
	return &LogServiceProvider{}
}

func (r *LogServiceProvider) Boot() {
	multiWriter := (core.NewLogger()).SetupWriter()
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter
}
