package app

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Error represents a structured error response.
type Error struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	FieldErrors map[string]string `json:"field_errors,omitempty"`
}

// Envelope wraps successful data or an error.
type Envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Error *Error      `json:"error,omitempty"`
}

// AbortError records an error and aborts the handler. The response will be
// rendered by the Errors middleware.
func AbortError(c *gin.Context, status int, code, message string, fields map[string]string) {
	c.Set("app_error", &Error{Code: code, Message: message, FieldErrors: fields})
	c.AbortWithStatus(status)
}

// Errors emits a JSON error envelope and structured log entry when an error
// was recorded via AbortError.
func Errors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		v, ok := c.Get("app_error")
		if !ok {
			return
		}
		err, ok := v.(*Error)
		if !ok {
			return
		}
		status := c.Writer.Status()
		logger := log.Ctx(c.Request.Context()).Error().Str("code", err.Code)
		if err.FieldErrors != nil {
			for k, v := range err.FieldErrors {
				logger = logger.Str("field_"+k, v)
			}
		}
		logger.Msg(err.Message)
		c.JSON(status, Envelope{Error: err})
	}
}
