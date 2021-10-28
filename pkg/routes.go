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

type RouteListenerMapping struct {
	Route    string
	Listener *CompiledListener
}

type MountRoutesResult struct {
	listenersMap map[string]*CompiledListener
}

func MountRoutes(engine *gin.Engine, config *Config, listenerIdPrefix string) (*MountRoutesResult, error) {
	storageCache := new(sync.Map)

	listenersMap := make(map[string]*CompiledListener)

	for route, listenerConfig := range config.Listeners {
		log := logrus.WithField("listener", route)

		listener, err := compileListener(&config.Defaults, listenerConfig, route, false, storageCache)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to compile listener for route %s", route)
		}
		handler := getGinListenerHandler(listener)
		mountedMethods := mountRoutesForListener(engine, listener, route, handler)

		// Populate the map of listeners so that we can later lookup listeners to perform async executions
		for _, m := range mountedMethods {
			id := spew.Sprintf("%s%s_%s", listenerIdPrefix, route, m)

			listener.SetId(id)
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.WithField("id", id).Debug("set listener id")
			}

			listenersMap[id] = listener
		}

		for _, plugin := range listener.plugins {
			if plugin, ok := plugin.(PluginHookMountRoutes); ok {
				plugin.HookMountRoutes(engine)
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

	return &MountRoutesResult{
		listenersMap,
	}, nil
}

func mountRoutesForListener(engine *gin.Engine, listener *CompiledListener, route string, handler gin.HandlerFunc) []string {
	var methods []string

	if len(listener.config.Methods) == 0 {
		methods = []string{http.MethodGet, http.MethodPost}
	} else {
		methods = listener.config.Methods
	}

	for _, method := range methods {
		var handlers []gin.HandlerFunc

		for _, plugin := range listener.plugins {
			if plugin, ok := plugin.(PluginHookGetMiddlewares); ok {
				handlers = append(handlers, plugin.HookGetMiddlewares(method)...)
			}
		}

		handlers = append(handlers, handler)
		engine.Handle(method, route, handlers...)
	}

	logrus.WithField("methods", methods).Infof("mounted route %s", route)

	return methods
}

func (r *MountRoutesResult) PluginsStart() error {
	var allPlugins []PluginInterface

	for _, listener := range r.listenersMap {
		plugins := listener.Plugins()
		for _, plugin := range plugins {
			allPlugins = append(allPlugins, plugin)
		}
	}

	for _, plugin := range allPlugins {
		if plugin, ok := plugin.(PluginConfigValidateCheckOtherPlugins); ok {
			var otherPlugins []PluginInterface
			for _, other := range allPlugins {
				if other == plugin {
					continue
				}
				otherPlugins = append(otherPlugins, other)
			}
			if err := plugin.ValidateCheckOtherPlugins(otherPlugins); err != nil {
				return errors.WithMessage(err, "failed to validate plugin against other plugins")
			}
		}
	}

	// For listeners which mount multiple methods, keep track of which plugins
	// have been started, to prevent double OnStart()
	var startedPlugins []PluginLifecycle

	for _, listener := range r.listenersMap {
		plugins := listener.Plugins()
		for _, plugin := range plugins {

			if plugin, ok := plugin.(PluginLifecycle); ok {
				// Do not start the same plugin twice
				found := false
				for _, other := range startedPlugins {
					if other == plugin {
						found = true
						break
					}
				}
				if found {
					continue
				}

				if err := plugin.OnStart(); err != nil {
					return errors.WithMessage(err, "failed to start plugin")
				}
				startedPlugins = append(startedPlugins, plugin)
			}
		}
	}

	return nil
}

func (r *MountRoutesResult) PluginsStop() {
	// For listeners which mount multiple methods, keep track of which plugins
	// have been stopped, to prevent double OnStop()
	var stoppedPlugins []PluginLifecycle

	for _, listener := range r.listenersMap {
		plugins := listener.Plugins()
		for _, plugin := range plugins {
			if plugin, ok := plugin.(PluginLifecycle); ok {
				// Do not start the same plugin twice
				found := false
				for _, other := range stoppedPlugins {
					if other == plugin {
						found = true
						break
					}
				}
				if found {
					continue
				}

				plugin.OnStop()
				stoppedPlugins = append(stoppedPlugins, plugin)
			}
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

		ctxHandled, response, err := listener.HandleRequest(c, args, nil)
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
