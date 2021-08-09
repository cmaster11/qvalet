package pkg

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
)

type GoToExec struct {
	config *Config
}

func NewGoToExec(config *Config) *GoToExec {
	return &GoToExec{
		config: config,
	}
}

func (gte *GoToExec) MountRoutes(engine *gin.Engine) {
	for route, listenerConfig := range gte.config.Listeners {
		listener := gte.compileListener(listenerConfig, route)
		handler := gte.getGinListenerHandler(listener)

		if len(listenerConfig.Methods) == 0 {
			engine.GET(route, handler)
			engine.POST(route, handler)
		} else {
			for _, method := range listenerConfig.Methods {
				engine.Handle(method, route, handler)
			}
		}
	}
}

func (gte *GoToExec) getGinListenerHandler(listener *CompiledListener) gin.HandlerFunc {
	return func(c *gin.Context) {
		args := make(map[string]interface{})

		// Use route params, if any
		for _, param := range c.Params {
			args[param.Key] = param.Value
		}

		if c.Request.Method != http.MethodGet {
			b := binding.Default(c.Request.Method, c.ContentType())
			if b == binding.Form || b == binding.FormMultipart {
				queryMap := make(map[string][]string)
				if err := c.ShouldBindWith(&queryMap, b); err != nil {
					c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request form body"))
					return
				}
				for key, vals := range queryMap {
					if len(vals) > 0 {
						args[key] = vals[len(vals)-1]
					} else {
						args[key] = true
					}

					args["_form_"+key] = vals
				}
			} else {
				if err := c.ShouldBindWith(&args, b); err != nil {
					c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request body"))
					return
				}
			}
		}

		// Always bind query
		{
			queryMap := make(map[string][]string)
			if err := c.ShouldBindQuery(&queryMap); err != nil {
				c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request query"))
				return
			}
			for key, vals := range queryMap {
				if len(vals) > 0 {
					args[key] = vals[len(vals)-1]
				} else {
					args[key] = true
				}

				args["_query_"+key] = vals
			}
		}

		out, err := listener.ExecCommand(args)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.WithMessagef(err, "failed to execute listener %s", listener.route))
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"output": out,
		})
	}
}
