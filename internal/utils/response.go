package utils

import "github.com/gin-gonic/gin"

type Envelope struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func OK(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Success: true, Data: data})
}

func ResponseFailed(c *gin.Context, status int, code, message string, details ...any) {
	var detail any
	if len(details) > 0 {
		detail = details[0]
	}
	c.AbortWithStatusJSON(status, Envelope{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message, Details: detail},
	})
}
