package utils

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
