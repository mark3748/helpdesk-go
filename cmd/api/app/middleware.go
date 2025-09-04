package app

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// RequestID assigns a UUID to each request and stores it in the context and response headers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.New().String()
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		logger := log.With().Str("request_id", id).Logger()
		ctx := logger.WithContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RateLimit applies a token bucket limiter to incoming requests.
func RateLimit(l *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// Logger emits a structured log entry for each request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)
		logger := log.Ctx(c.Request.Context()).Info()
		logger.Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", dur).
			Msg("request")
	}
}
