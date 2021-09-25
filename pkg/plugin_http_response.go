package pkg

import (
	"fmt"
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
	Headers map[string]string `mapstructure:"headers"`

	// The template for the desired StatusCode. If empty, 200 will be used.
	// NOTE that the result will be trimmed, so you don't have to worry about
	// white spaces and newlines issues.
	StatusCode string `mapstructure:"statusCode"`

	// TODO: support other options
}

/// [config]
// @formatter:on

type PluginHTTPResponse struct {
	listener *CompiledListener

	config *PluginHTTPResponseConfig

	headerTemplates    map[string]*Template
	statusCodeTemplate *Template
}

func (c *PluginHTTPResponseConfig) NewPlugin(listener *CompiledListener) (Plugin, error) {
	tplFuncs := listener.TplFuncMap()

	plugin := &PluginHTTPResponse{
		listener: listener,
		config:   c,
	}

	{
		headerTemplates := make(map[string]*Template)
		for key, str := range c.Headers {
			tpl, err := ParseTemplate(fmt.Sprintf("header-%s", key), str, tplFuncs)
			if err != nil {
				listener.Logger().WithError(err).WithField("header", key).WithField("template", tpl).Fatal("failed to parse listener plugin http response header template")
			}
			headerTemplates[key] = tpl
		}
		plugin.headerTemplates = headerTemplates
	}

	if c.StatusCode != "" {
		tpl, err := ParseTemplate(fmt.Sprintf("status-code"), c.StatusCode, tplFuncs)
		if err != nil {
			listener.Logger().WithError(err).WithField("template", tpl).Fatal("failed to parse listener plugin http response status code template")
		}
		plugin.statusCodeTemplate = tpl
	}

	return plugin, nil
}

func (c *PluginHTTPResponseConfig) IsUnique() bool {
	return false
}

func (p PluginHTTPResponse) Clone(newListener *CompiledListener) Plugin {
	funcMap := newListener.TplFuncMap()

	newPlugin := &PluginHTTPResponse{
		listener: newListener,
		config:   p.config,
	}

	if p.statusCodeTemplate != nil {
		tpl, err := p.statusCodeTemplate.Clone()
		if err == nil {
			tpl.Funcs(funcMap)
		}
		newPlugin.statusCodeTemplate = tpl
	}

	{
		headerTemplates := make(map[string]*Template)
		for key, tpl := range p.headerTemplates {
			tpl, err := tpl.Clone()
			if err == nil {
				tpl.Funcs(funcMap)
			}
			headerTemplates[key] = tpl
		}
		newPlugin.headerTemplates = headerTemplates
	}

	return newPlugin
}

func (p PluginHTTPResponse) HookOutput(c *gin.Context, args map[string]interface{}, listenerResponse *ListenerResponse) (handled bool, err error) {
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
			c.Header(key, out)
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

	c.JSON(statusCode, listenerResponse)

	handled = true
	return
}
