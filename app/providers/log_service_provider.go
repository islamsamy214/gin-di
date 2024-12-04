package providers

import (
	"web-app/app/services/core"

	"github.com/gin-gonic/gin"
)

type LogServiceProvider struct{}

func (l *LogServiceProvider) Boot() {
	// Boot the logger service
	multiWriter := (&core.Logger{}).SetupWriter()
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter
}
