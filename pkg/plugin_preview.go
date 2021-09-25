package pkg

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

var _ Plugin = (*PluginPreview)(nil)
var _ PluginHookMountRoutes = (*PluginPreview)(nil)
var _ PluginConfig = (*PluginPreviewConfig)(nil)

const pluginPreviewRouteDefault = "/preview"

// @formatter:off
/// [config]
type PluginPreviewConfig struct {
	// List of allowed authentication methods
	Auth []*AuthConfig `mapstructure:"auth" validate:"dive"`

	// Path to append, defaults to `/preview`
	Path *string

	// If true, the response will be formatted as YAML
	AsYAML bool
}

/// [config]
// @formatter:on

func (c *PluginPreviewConfig) NewPlugin(listener *CompiledListener) (Plugin, error) {
	return &PluginPreview{
		config: c,
	}, nil
}

func (c *PluginPreviewConfig) IsUnique() bool {
	return false
}

type PluginPreview struct {
	config *PluginPreviewConfig
}

func (p *PluginPreview) Clone(newListener *CompiledListener) Plugin {
	return p
}

func (p *PluginPreview) HookMountRoutes(engine *gin.Engine, listener *CompiledListener) {
	route := pluginPreviewRouteDefault
	if p.config.Path != nil {
		route = *p.config.Path
	}

	handler := func(c *gin.Context) {
		handled, args := prepareListenerRequestHandling(c, p.config.Auth)
		if handled {
			return
		}

		toStore := make(map[string]interface{})

		preparedExecutionResult, err := listener.clone().prepareExecution(args, toStore)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to prepare command execution"))
		}

		if p.config.AsYAML {
			yml, _ := yaml.MarshalWithOptions(
				preparedExecutionResult,
				yaml.UseLiteralStyleIfMultiline(true),
			)
			c.String(http.StatusOK, string(yml))
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, preparedExecutionResult)
	}

	mountRoutesByMethod(engine, listener.config.Methods, fmt.Sprintf("%s%s", listener.route, route), handler)
}
