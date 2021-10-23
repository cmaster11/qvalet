package pkg

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

var _ PluginInterface = (*PluginPreview)(nil)
var _ PluginHookMountRoutes = (*PluginPreview)(nil)
var _ PluginConfig = (*PluginPreviewConfig)(nil)

const pluginPreviewRouteDefault = "/preview"

// @formatter:off
/// [config]
type PluginPreviewConfig struct {
	// List of allowed authentication methods, defaults to the listener ones
	Auth []*AuthConfig `mapstructure:"auth" validate:"dive"`

	// Route to append, defaults to `/preview`
	Route *string `mapstructure:"route"`

	// If true, the response will be formatted as YAML
	AsYAML bool `mapstructure:"asYaml"`
}

/// [config]
// @formatter:on

func (c *PluginPreviewConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	return &PluginPreview{
		NewPluginBase("preview"),
		c,
		listener,
	}, nil
}

func (c *PluginPreviewConfig) IsUnique() bool {
	return false
}

type PluginPreview struct {
	PluginBase

	config   *PluginPreviewConfig
	listener *CompiledListener
}

func (p *PluginPreview) Clone(_ *CompiledListener) (PluginInterface, error) {
	return p, nil
}

func (p *PluginPreview) HookMountRoutes(engine *gin.Engine) {
	route := pluginPreviewRouteDefault
	if p.config.Route != nil {
		route = *p.config.Route
	}

	handler := func(c *gin.Context) {
		authConfig := p.config.Auth
		if authConfig == nil {
			authConfig = p.listener.config.Auth
		}

		handled, args := prepareListenerRequestHandling(c, authConfig)
		if handled {
			return
		}

		toStore := make(map[string]interface{})

		listenerClone, err := p.listener.clone()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.WithMessage(err, "failed to clone listener"))
			return
		}
		preparedExecutionResult, handledResult, err := listenerClone.prepareExecution(args, toStore)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to prepare command execution"))
			return
		}

		var toReturn interface{}
		toReturn = preparedExecutionResult
		if handledResult != nil {
			toReturn = handledResult
		}

		if p.config.AsYAML {
			yml, _ := yaml.MarshalWithOptions(
				toReturn,
				yaml.UseLiteralStyleIfMultiline(true),
			)
			c.String(http.StatusOK, string(yml))
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, toReturn)
	}

	mountRoutesByMethod(engine, p.listener.config.Methods, fmt.Sprintf("%s%s", p.listener.route, route), handler)
}
