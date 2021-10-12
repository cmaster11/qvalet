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
	// List of allowed authentication methods
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
	}, nil
}

func (c *PluginPreviewConfig) IsUnique() bool {
	return false
}

type PluginPreview struct {
	PluginBase

	config *PluginPreviewConfig
}

func (p *PluginPreview) Clone(newListener *CompiledListener) (PluginInterface, error) {
	return p, nil
}

func (p *PluginPreview) HookMountRoutes(engine *gin.Engine, listener *CompiledListener) {
	route := pluginPreviewRouteDefault
	if p.config.Route != nil {
		route = *p.config.Route
	}

	handler := func(c *gin.Context) {
		handled, args := prepareListenerRequestHandling(c, p.config.Auth)
		if handled {
			return
		}

		toStore := make(map[string]interface{})

		listenerClone, err := listener.clone()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.WithMessage(err, "failed to clone listener"))
		}
		preparedExecutionResult, handledResult, err := listenerClone.prepareExecution(args, toStore)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to prepare command execution"))
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

	mountRoutesByMethod(engine, listener.config.Methods, fmt.Sprintf("%s%s", listener.route, route), handler)
}