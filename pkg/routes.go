package pkg

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const keyAuthDefaultHTTPBasicUser = "gte"
const keyAuthApiKeyQuery = "__gteApiKey"

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
		log := logrus.WithField("listener", route)

		listener := gte.compileListener(listenerConfig, route, false)
		handler := gte.getGinListenerHandler(listener)

		if len(listener.config.Methods) == 0 {
			engine.GET(route, handler)
			engine.POST(route, handler)
		} else {
			for _, method := range listener.config.Methods {
				engine.Handle(method, route, handler)
			}
		}

		log.WithFields(logrus.Fields{
			"config": spew.Sdump(listener.config),
		}).Debug("added listener")
	}
}

func (gte *GoToExec) getGinListenerHandler(listener *CompiledListener) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := gte.verifyAuth(c, listener); err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

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
			if listener.errorHandler != nil {
				handler := listener.errorHandler
				// Trigger a command on error
				onErrorArgs := map[string]interface{}{
					"route":  listener.route,
					"error":  err.Error(),
					"output": out,
					"args":   args,
				}
				_, err := handler.ExecCommand(onErrorArgs)
				if err != nil {
					handler.log.WithError(err).Error("failed to execute error listener")
				} else {
					handler.log.Info("executed error listener")
				}
			}

			c.AbortWithError(http.StatusInternalServerError, errors.WithMessagef(err, "failed to execute listener %s", listener.route))
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"output": out,
		})
	}
}

func (gte *GoToExec) verifyAuth(c *gin.Context, listener *CompiledListener) error {
	if len(listener.config.Auth) == 0 {
		return nil
	}

	// Auth check
	found := false

	for _, auth := range listener.config.Auth {

		// Basic HTTP authentication
		if auth.BasicAuth {
			authUser := auth.BasicAuthUser
			if authUser == "" {
				// Default user for basic auth
				authUser = keyAuthDefaultHTTPBasicUser
			}
			// Check if there is any basic auth
			if username, password, ok := c.Request.BasicAuth(); ok {
				if username == authUser {
					for _, apiKey := range auth.ApiKeys {
						if password == apiKey {
							found = true
							goto afterAuth
						}
					}
				}
			}
		}

		// Url query authentication
		if auth.QueryAuth {
			queryKey := auth.QueryAuthKey
			if queryKey == "" {
				queryKey = keyAuthApiKeyQuery
			}
			apiKeyQuery := c.Query(queryKey)
			for _, apiKey := range auth.ApiKeys {
				if apiKeyQuery == apiKey {
					found = true
					goto afterAuth
				}
			}
		}

		// Header authentication
		if len(auth.AuthHeaders) > 0 {
			for _, authHeader := range auth.AuthHeaders {
				headerValue := c.GetHeader(authHeader.Header)
				for _, apiKey := range auth.ApiKeys {
					if headerValue == apiKey {
						found = true
						goto afterAuth
					}
				}
			}
		}
	}

afterAuth:

	if !found {
		return errors.New("bad auth")
	}

	return nil
}
