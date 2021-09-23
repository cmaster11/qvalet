package plugins

import (
	"fmt"

	"gotoexec/pkg/plugins/snshttp"
	"gotoexec/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var _ Plugin = (*PluginAWSSNS)(nil)
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

func (c *PluginAWSSNSConfig) NewPlugin() (Plugin, error) {
	return NewPluginAWSSNS(c), nil
}

func (c *PluginAWSSNSConfig) IsUnique() bool {
	return true
}

type PluginAWSSNS struct {
	config     *PluginAWSSNSConfig
	snsHandler *snshttp.SNSHandler
}

func (p *PluginAWSSNS) HookPreExecute(args map[string]interface{}) (map[string]interface{}, error) {
	return args, nil
}

func NewPluginAWSSNS(config *PluginAWSSNSConfig) *PluginAWSSNS {
	var options []snshttp.Option

	if config.BasicAuth != nil {
		options = append(options, snshttp.WithAuthentication(config.BasicAuth.Username, config.BasicAuth.Password))
	}

	snsHandler := snshttp.NewSNSHTTPHandler(options...)

	plugin := &PluginAWSSNS{
		config:     config,
		snsHandler: snsHandler,
	}

	return plugin
}

func (p *PluginAWSSNS) MountRoutes(engine *gin.Engine, listenerRoute string, listenerHandler func(args map[string]interface{}) (interface{}, error)) {
	engine.POST(fmt.Sprintf("%s/sns", listenerRoute), p.snsHandler.GetSNSRequestHandler(func(c *gin.Context, notification *snshttp.SNSNotification) error {

		args := make(map[string]interface{})
		if err := utils.DecodeStructJSONToMap(notification, &args); err != nil {
			return errors.WithMessage(err, "failed to decode sns notification struct to map")
		}

		_, err := listenerHandler(args)
		return err
	}))
}
