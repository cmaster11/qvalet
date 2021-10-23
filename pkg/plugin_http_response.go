package pkg

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var _ PluginHookOutput = (*PluginHTTPResponse)(nil)
var _ PluginConfig = (*PluginHTTPResponseConfig)(nil)

const keyPluginHTTPResponseListenerResponse = "__gteResult"

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
}

/// [config]
// @formatter:on

type PluginHTTPResponse struct {
	PluginBase

	listener *CompiledListener

	config *PluginHTTPResponseConfig

	headerTemplates    map[string]*ListenerTemplate
	statusCodeTemplate *ListenerTemplate
}

func (c *PluginHTTPResponseConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	plugin := &PluginHTTPResponse{
		NewPluginBase("http-response"),
		listener,
		c,
		c.Headers,
		c.StatusCode,
	}

	return plugin, nil
}

func (c *PluginHTTPResponseConfig) IsUnique() bool {
	return false
}

func (p PluginHTTPResponse) Clone(newListener *CompiledListener) (PluginInterface, error) {
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

func (p PluginHTTPResponse) HookOutput(writeOnlyContext *gin.Context, args map[string]interface{}, listenerResponse *ListenerResponse) (handled bool, err error) {
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
