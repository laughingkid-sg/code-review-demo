package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs incoming HTTP requests with structured fields.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		entry := map[string]any{
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
		}

		if errorMessage != "" {
			entry["error"] = errorMessage
		}

		payload, err := json.Marshal(entry)
		if err != nil {
			log.Printf("[GIN] %s | %3d | %13v | %15s | %-7s %s",
				time.Now().Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
			)
			return
		}

		log.Print(string(payload))
	}
}
