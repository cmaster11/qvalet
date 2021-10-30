package pkg

import (
	"net/http"
	"strconv"
	"strings"

	"qvalet/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var _ PluginHookOutput = (*PluginHTTPResponse)(nil)
var _ PluginHookMountRoutes = (*PluginHTTPResponse)(nil)
var _ PluginHookGetMiddlewares = (*PluginHTTPResponse)(nil)
var _ PluginConfig = (*PluginHTTPResponseConfig)(nil)

const keyPluginHTTPResponseListenerResponse = "__qvResult"

// @formatter:off
/// [config]
type PluginHTTPResponseConfig struct {
	// The Headers map can contain any headers you want to set in the HTTP response.
	// The header value can be a template.
	// NOTE that the result will be trimmed, so you don't have to worry about
	// white spaces and newlines issues.
	Headers map[string]*ListenerTemplate `mapstructure:"headers"`

	// The template for the desired StatusCode. If empty, 200 will be used.
	// NOTE that the result will be trimmed, so you don't have to worry about
	// white spaces and newlines issues.
	StatusCode *ListenerTemplate `mapstructure:"statusCode"`

	// If you want to allow CORS, you can specify the configuration
	// here
	CORS *CORSConfig `mapstructure:"cors"`
}

/// [config]
/// [config-cors]
type CORSConfig struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string `mapstructure:"allowedOrigins"`

	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Defaults to the list of methods accepted by the listener.
	AllowedMethods []string `mapstructure:"allowedMethods"`

	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string `mapstructure:"allowedHeaders"`

	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string `mapstructure:"exposedHeaders"`

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int `mapstructure:"maxAge"`

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `mapstructure:"allowCredentials"`
}

/// [config-cors]
// @formatter:on

type PluginHTTPResponse struct {
	PluginBase

	listener *CompiledListener

	config *PluginHTTPResponseConfig

	headerTemplates    map[string]*ListenerTemplate
	statusCodeTemplate *ListenerTemplate
	corsHandler        gin.HandlerFunc
}

func (c *PluginHTTPResponseConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	var corsHandler gin.HandlerFunc
	if c.CORS != nil {
		pluginCORSConfig := c.CORS
		allowPassthrough := false
		if utils.StringSliceContains(listener.config.Methods, http.MethodOptions) {
			allowPassthrough = true
		}
		corsOptions := cors.Options{
			AllowedOrigins:         pluginCORSConfig.AllowedOrigins,
			AllowOriginFunc:        nil,
			AllowOriginRequestFunc: nil,
			AllowedMethods:         listener.config.Methods,
			AllowedHeaders:         pluginCORSConfig.AllowedHeaders,
			ExposedHeaders:         pluginCORSConfig.ExposedHeaders,
			MaxAge:                 pluginCORSConfig.MaxAge,
			AllowCredentials:       pluginCORSConfig.AllowCredentials,
			OptionsPassthrough:     allowPassthrough,
			Debug:                  logrus.IsLevelEnabled(logrus.DebugLevel),
		}

		corsInstance := cors.New(corsOptions)
		if corsOptions.Debug {
			corsInstance.Log = logrus.WithField("logger", "cors")
		}
		corsHandler = func(ctx *gin.Context) {
			corsInstance.HandlerFunc(ctx.Writer, ctx.Request)
			if !corsOptions.OptionsPassthrough &&
				ctx.Request.Method == http.MethodOptions &&
				ctx.GetHeader("Access-Control-Request-Method") != "" {
				// Abort processing next Gin middlewares.
				ctx.AbortWithStatus(http.StatusOK)
			}
		}
	}

	plugin := &PluginHTTPResponse{
		NewPluginBase("http-response"),
		listener,
		c,
		c.Headers,
		c.StatusCode,
		corsHandler,
	}

	return plugin, nil
}

func (c *PluginHTTPResponseConfig) IsUnique() bool {
	return false
}

func (p *PluginHTTPResponse) Clone(newListener *CompiledListener) (PluginInterface, error) {
	newPlugin := &PluginHTTPResponse{
		PluginBase: p.PluginBase,
		listener:   newListener,
		config:     p.config,
	}

	if p.statusCodeTemplate != nil {
		tpl, err := p.statusCodeTemplate.CloneForListener(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone status template")
		}
		newPlugin.statusCodeTemplate = tpl
	}

	{
		headerTemplates := make(map[string]*Template)
		for key, tpl := range p.headerTemplates {
			tpl, err := tpl.CloneForListener(newListener)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to clone header template")
			}
			headerTemplates[key] = tpl
		}
		newPlugin.headerTemplates = headerTemplates
	}

	return newPlugin, nil
}

func (p *PluginHTTPResponse) HookOutput(writeOnlyContext *gin.Context, args map[string]interface{}, listenerResponse *ListenerResponse) (handled bool, err error) {
	newArgs := make(map[string]interface{})

	for key, val := range args {
		newArgs[key] = val
	}
	newArgs[keyPluginHTTPResponseListenerResponse] = listenerResponse

	for key, tpl := range p.headerTemplates {
		out, err := tpl.Execute(newArgs)
		if err != nil {
			err := errors.WithMessage(err, "failed to execute plugin http response header template")
			p.listener.Logger().WithField("header", key).WithError(err).Error("error")
			return false, err
		}

		out = strings.TrimSpace(out)

		if out != "" {
			writeOnlyContext.Header(key, out)
		}
	}

	statusCode := http.StatusOK

	if p.statusCodeTemplate != nil {
		out, err := p.statusCodeTemplate.Execute(newArgs)
		if err != nil {
			err := errors.WithMessage(err, "failed to execute plugin http response status code template")
			p.listener.Logger().WithError(err).Error("error")
			return false, err
		}

		trimmed := strings.TrimSpace(out)
		if trimmed == "" {
			statusCode = http.StatusOK
		} else {
			parsed, err := strconv.ParseInt(trimmed, 10, 32)
			if err != nil {
				err := errors.WithMessage(err, "failed to parse plugin http response status code")
				p.listener.Logger().WithField("statusCode", trimmed).WithError(err).Error("error")
				return false, err
			}
			statusCode = int(parsed)
		}
	}

	writeOnlyContext.JSON(statusCode, listenerResponse)

	handled = true
	return
}

func (p *PluginHTTPResponse) HookMountRoutes(engine *gin.Engine) {
	if p.corsHandler != nil && !utils.StringSliceContains(p.listener.config.Methods, http.MethodOptions) {
		engine.OPTIONS(p.listener.route, p.corsHandler)
	}
}

func (p *PluginHTTPResponse) HookGetMiddlewares(method string) []gin.HandlerFunc {
	if p.corsHandler != nil {
		return []gin.HandlerFunc{p.corsHandler}
	}
	return nil
}
