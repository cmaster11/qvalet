package utils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	GinContextDoNotLogEntry string = "GinContextDoNotLogEntry"
)

type RequestError struct {
	StatusCode int
	Err        error
}

func (r *RequestError) Error() string {
	return fmt.Sprintf("[%d] %v", r.StatusCode, r.Err)
}

type WrappedRequestFn func(c *gin.Context) (interface{}, error)

func WrapRequest(fn WrappedRequestFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := fn(c)

		if c.IsAborted() {
			// NOOP
			return
		}

		if err != nil {
			if err, ok := err.(*RequestError); ok {
				statusCode := err.StatusCode
				if statusCode <= 0 {
					statusCode = http.StatusInternalServerError
				}
				c.AbortWithError(statusCode, err)
				return
			}

			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if result != nil {
			if str, ok := result.(string); ok {
				contentType := http.DetectContentType([]byte(str))
				c.Header("Content-Type", contentType)
			}

			c.AbortWithStatusJSON(http.StatusOK, result)
			return
		}

		c.AbortWithStatus(http.StatusOK)
	}
}

func GetGinLoggerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		if c.GetBool(GinContextDoNotLogEntry) {
			return
		}

		statusCode := c.Writer.Status()

		clientIP := c.ClientIP()
		method := c.Request.Method

		if statusCode == http.StatusOK && method == http.MethodOptions {
			// Do not log useless entries
			return
		}

		elapsedMS := time.Now().Sub(start).Milliseconds()

		comment := c.Errors.ByType(gin.ErrorTypePrivate).String()
		userAgent := c.Request.UserAgent()

		logrus.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"path":       path,
			"elapsedMS":  elapsedMS,
			"clientIP":   clientIP,
			"method":     method,
			"comment":    comment,
			"userAgent":  userAgent,
		}).Info(fmt.Sprintf("[GIN] %3d | %13vms | %s %-7s | %s",
			statusCode,
			elapsedMS,
			method,
			path,
			comment,
		))

	}
}
