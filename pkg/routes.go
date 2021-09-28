package pkg

import (
	"net/http"
	"regexp"
	"sync"

	"gotoexec/pkg/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const keyAuthDefaultHTTPBasicUser = "gte"
const keyAuthApiKeyQuery = "__gteApiKey"

func MountRoutes(engine *gin.Engine, config *Config) {
	storageCache := new(sync.Map)

	for route, listenerConfig := range config.Listeners {
		log := logrus.WithField("listener", route)

		listener := compileListener(&config.Defaults, listenerConfig, route, false, storageCache)
		handler := getGinListenerHandler(listener)
		mountRoutesByMethod(engine, listener.config.Methods, route, handler)

		for _, plugin := range listener.plugins {
			if plugin, ok := plugin.(PluginHookMountRoutes); ok {
				plugin.HookMountRoutes(engine, listener)
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

func mountRoutesByMethod(engine *gin.Engine, methods []string, route string, handler gin.HandlerFunc) {
	if len(methods) == 0 {
		engine.GET(route, handler)
		engine.POST(route, handler)
	} else {
		for _, method := range methods {
			engine.Handle(method, route, handler)
		}
	}
}

// @formatter:off
/// [listener-response]
type ListenerResponse struct {
	*ExecCommandResult
	Storage            *StorageEntry     `json:"storage,omitempty"`
	Error              *string           `json:"error,omitempty"`
	ErrorHandlerResult *ListenerResponse `json:"errorHandlerResult,omitempty"`
}

/// [listener-response]
// @formatter:on

var regexListenerRouteCleaner = regexp.MustCompile(`[\W]`)

func getGinListenerHandler(listener *CompiledListener) gin.HandlerFunc {
	return func(c *gin.Context) {
		handled, args := prepareListenerRequestHandling(c, listener.config.Auth)
		if handled {
			return
		}

		ctxHandled, response, err := listener.HandleRequest(c, args)
		if ctxHandled {
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func prepareListenerRequestHandling(
	c *gin.Context,
	authConfigs []*AuthConfig,
) (bool, map[string]interface{}) {
	if err := verifyAuth(c, authConfigs); err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return true, nil
	}

	args, err := utils.ExtractArgsFromGinContext(c)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to extract args from request"))
		return true, nil
	}
	return false, args
}
