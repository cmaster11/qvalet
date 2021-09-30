package pkg

import (
	"fmt"

	snshttp2 "gotoexec/pkg/snshttp"
	"gotoexec/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var _ PluginInterface = (*PluginAWSSNS)(nil)
var _ PluginHookMountRoutes = (*PluginAWSSNS)(nil)
var _ PluginConfig = (*PluginAWSSNSConfig)(nil)

// @formatter:off
/// [config]
type PluginAWSSNSConfig struct {
	// If defined, the SNS subscription url MUST contain the specified
	// username and password pair, e.g. https://user:pwd@mydomain.com/test/sns
	// NOTE: if basic auth is defined, the SNS subscription MUST be of HTTPS type.
	BasicAuth *PluginAWSSNSConfigBasicAuth `mapstructure:"basicAuth"`
}

type PluginAWSSNSConfigBasicAuth struct {
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
}

/// [config]
// @formatter:on

func (c *PluginAWSSNSConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	var options []snshttp2.Option

	if c.BasicAuth != nil {
		options = append(options, snshttp2.WithAuthentication(c.BasicAuth.Username, c.BasicAuth.Password))
	}

	snsHandler := snshttp2.NewSNSHTTPHandler(options...)

	plugin := &PluginAWSSNS{
		NewPluginBase("awssns"),
		c,
		snsHandler,
	}

	return plugin, nil
}

func (c *PluginAWSSNSConfig) IsUnique() bool {
	return true
}

type PluginAWSSNS struct {
	PluginBase

	config     *PluginAWSSNSConfig
	snsHandler *snshttp2.SNSHandler
}

func (p *PluginAWSSNS) Clone(newListener *CompiledListener) (PluginInterface, error) {
	return p, nil
}

func (p *PluginAWSSNS) HookMountRoutes(engine *gin.Engine, listener *CompiledListener) {
	engine.POST(fmt.Sprintf("%s/sns", listener.route), p.snsHandler.GetSNSRequestHandler(func(c *gin.Context, notification *snshttp2.SNSNotification) error {

		args := make(map[string]interface{})
		if err := utils.DecodeStructJSONToMap(notification, &args); err != nil {
			return errors.WithMessage(err, "failed to decode sns notification struct to map")
		}

		_, _, err := listener.HandleRequest(c, args, nil)
		return err
	}))
}
