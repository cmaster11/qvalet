package pkg

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const keyAuthDefaultHTTPBasicUser = "gte"
const keyAuthApiKeyQuery = "__gteApiKey"
const keyArgsHeadersKey = "__gteHeaders"

func MountRoutes(engine *gin.Engine, config *Config) {
	for route, listenerConfig := range config.Listeners {
		log := logrus.WithField("listener", route)

		listener := compileListener(&config.Defaults, listenerConfig, route, false)
		handler := getGinListenerHandler(listener)

		if len(listener.config.Methods) == 0 {
			engine.GET(route, handler)
			engine.POST(route, handler)
		} else {
			for _, method := range listener.config.Methods {
				engine.Handle(method, route, handler)
			}
		}

		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			log.WithFields(logrus.Fields{
				"config": spew.Sdump(listener.config),
			}).Debug("added listener")
		} else {
			log.Info("added listener")
		}
	}
}

type ListenerResponse struct {
	*ExecCommandResult
	Storage            *StorageEntry     `json:"storage,omitempty"`
	Error              *string           `json:"error,omitempty"`
	ErrorHandlerResult *ListenerResponse `json:"errorHandlerResult,omitempty"`
}

var regexListenerRouteCleaner = regexp.MustCompile(`[\W]`)

func getGinListenerHandler(listener *CompiledListener) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := verifyAuth(c, listener); err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		// Keep track of what to store
		toStore := make(map[string]interface{})

		args := make(map[string]interface{})

		// Use route params, if any
		for _, param := range c.Params {
			args[param.Key] = param.Value
		}

		// Add headers to args
		{
			headerMap := make(map[string]interface{})
			for k := range c.Request.Header {
				headerMap[strings.ToLower(k)] = c.GetHeader(k)
			}
			args[keyArgsHeadersKey] = headerMap
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

		if listener.storager != nil && listener.config.Storage.StoreArgs() {
			toStore["args"] = args
		}

		out, err := listener.ExecCommand(args, toStore)
		if err != nil {
			err := errors.WithMessagef(err, "failed to execute listener %s", listener.route)
			response := &ListenerResponse{
				ExecCommandResult: out,
				Error:             stringPtr(err.Error()),
			}

			var errorHandlerResult *ListenerResponse
			if listener.errorHandler != nil {
				errorHandler := listener.errorHandler

				errorHandlerResult = &ListenerResponse{}

				toStoreOnError := make(map[string]interface{})

				// Trigger a command on error
				onErrorArgs := map[string]interface{}{
					"route":  listener.route,
					"error":  err.Error(),
					"output": out,
					"args":   args,
				}

				if errorHandler.storager != nil && errorHandler.config.Storage.StoreArgs() {
					toStoreOnError["args"] = args
				}

				errorHandlerExecCommandResult, err := errorHandler.ExecCommand(onErrorArgs, toStoreOnError)
				errorHandlerResult.ExecCommandResult = errorHandlerExecCommandResult
				if err != nil {
					errorHandlerResult.Error = stringPtr(err.Error())
					errorHandler.log.WithError(err).Error("failed to execute error listener")
				} else {
					errorHandler.log.Info("executed error listener")
				}

				if errorHandler.storager != nil && len(toStoreOnError) > 0 {
					if entry := storePayload(
						errorHandler,
						toStoreOnError,
					); entry != nil {
						if errorHandler.config.ReturnStorage() {
							errorHandlerResult.Storage = entry
						}
					}
				}

				toStore["errorHandler"] = toStoreOnError
				if listener.storager != nil && len(toStore) > 0 {
					if entry := storePayload(
						listener,
						toStore,
					); entry != nil {
						if listener.config.ReturnStorage() {
							response.Storage = entry
						}
					}
				}

				response.ErrorHandlerResult = errorHandlerResult
			}

			c.JSON(http.StatusInternalServerError, response)
			return
		}

		response := &ListenerResponse{
			ExecCommandResult: out,
		}

		if listener.storager != nil && len(toStore) > 0 {
			if entry := storePayload(
				listener,
				toStore,
			); entry != nil {
				if listener.config.ReturnStorage() {
					response.Storage = entry
				}
			}
		}

		c.JSON(http.StatusOK, response)
	}
}
