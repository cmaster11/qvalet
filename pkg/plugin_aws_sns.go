package pkg

import (
	"fmt"

	"gotoexec/pkg/snshttp"
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
	var options []snshttp.Option

	if c.BasicAuth != nil {
		options = append(options, snshttp.WithAuthentication(c.BasicAuth.Username, c.BasicAuth.Password))
	}

	snsHandler := snshttp.NewSNSHTTPHandler(options...)

	plugin := &PluginAWSSNS{
		NewPluginBase("awssns"),
		c,
		listener,
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
	listener   *CompiledListener
	snsHandler *snshttp.SNSHandler
}

func (p *PluginAWSSNS) Clone(_ *CompiledListener) (PluginInterface, error) {
	return p, nil
}

func (p *PluginAWSSNS) HookMountRoutes(engine *gin.Engine) {
	engine.POST(fmt.Sprintf("%s/sns", p.listener.route), p.snsHandler.GetSNSRequestHandler(func(c *gin.Context, notification *snshttp.SNSNotification) error {

		args := make(map[string]interface{})
		if err := utils.DecodeStructJSONToMap(notification, &args); err != nil {
			return errors.WithMessage(err, "failed to decode sns notification struct to map")
		}

		_, _, err := p.listener.HandleRequest(c, args, nil)
		return err
	}))
}
