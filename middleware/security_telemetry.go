package middleware

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// RecordSecurityTelemetry records authenticated request metadata after all
// route middleware and handlers have run. It intentionally stores hashes and
// headers, never bearer or refresh token values.
func RecordSecurityTelemetry() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		userID := c.GetInt("id")
		if userID <= 0 {
			return
		}
		eventType := "request"
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v1beta/") {
			eventType = "api_request"
		}
		modelName := c.GetString("risk_model_name")
		channelID := c.GetInt("risk_channel_id")
		tokenID := c.GetInt("token_id")
		_ = model.RecordSecurityRequest(c, userID, eventType, c.Writer.Status(), time.Since(started).Milliseconds(), modelName, channelID, tokenID)
	}
}
