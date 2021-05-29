package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type GoToExec struct {
	config *Config
}

func (gte *GoToExec) mountRoutes(engine *gin.Engine)  {
	for path, listenerConfig := range gte.config.Listeners {

		listener := func(c *gin.Context) {
			args := make(map[string]interface{})

			if c.Request.Method != http.MethodGet {
				if err := c.Bind(&args); err != nil {
					return
				}
			}

			// Always bind query
			if err := c.MustBindWith(&args, binding.Query); err != nil {
				return
			}

			if err := gte.execCommand(listenerConfig, args); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.AbortWithStatus(http.StatusOK)
		}

		engine.GET(path, listener)
		engine.POST(path, listener)
		engine.PUT(path, listener)
		engine.DELETE(path, listener)
	}
}

func (gte *GoToExec) execCommand(config *Listener, args map[string]interface{}) error {
	return nil
}

