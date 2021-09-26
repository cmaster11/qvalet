package pkg

import (
	"reflect"

	"gotoexec/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Plugin interface {
	Clone(newListener *CompiledListener) Plugin
}

type PluginHookMountRoutes interface {
	// Called on initialization, allows plugins to mount additional routes
	HookMountRoutes(engine *gin.Engine, listener *CompiledListener)
}

type PluginHookPreExecute interface {
	// Called at runtime, before listener execution, allows alteration of args
	HookPreExecute(args map[string]interface{}) (map[string]interface{}, error)
}

type PluginHookOutput interface {
	// Called at runtime, after listener successful execution, allows alteration of output
	HookOutput(c *gin.Context, args map[string]interface{}, listenerResponse *ListenerResponse) (handled bool, err error)
}

type PluginConfig interface {
	// If true, a plugin cannot be declared more than once for a listener
	IsUnique() bool

	// Instantiates the plugin related to this config
	NewPlugin(listener *CompiledListener) (Plugin, error)
}

type PluginEntryConfig struct {

	// AWS SNS plugin, to auto-confirm AWS SNS subscriptions and handle SNS notifications
	AWSSNS *PluginAWSSNSConfig `mapstructure:"awsSNS"`

	// HTTP response plugin, to alter HTTP response headers, status code, etc...
	HTTPResponse *PluginHTTPResponseConfig `mapstructure:"httpResponse"`

	// Preview plugin, used to preview the command which will be executed
	Preview *PluginPreviewConfig `mapstructure:"preview"`

	// ---

	// Debug plugin, for testing
	Debug *PluginDebugConfig `mapstructure:"debug"`
}

func (config *PluginEntryConfig) ToPluginList(listener *CompiledListener) ([]Plugin, error) {
	var plugins []Plugin

	configEntry := reflect.ValueOf(config).Elem()
	for j := 0; j < configEntry.NumField(); j++ {
		field := configEntry.Field(j)
		if field.IsNil() {
			continue
		}
		configField, ok := field.Interface().(PluginConfig)
		if !ok {
			return nil, errors.New("failed to cast plugin entry field to PluginConfig")
		}

		plugin, err := configField.NewPlugin(listener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to initialize plugin")
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func init() {
	if err := utils.Validate.RegisterValidation("uniquePlugins", func(fl validator.FieldLevel) bool {
		field := fl.Field()

		if field.Kind() != reflect.Slice {
			logrus.Fatal("non-slice type passed for uniquePlugins validator")
		}

		var configs []PluginConfig

		// We want to check that plugins marked as unique, are really unique overall
		for i := 0; i < field.Len(); i++ {
			configEntry := field.Index(i).Elem()
			for j := 0; j < configEntry.NumField(); j++ {
				configField := configEntry.Field(j)
				if configField.IsNil() {
					continue
				}
				config, ok := configField.Interface().(PluginConfig)
				if !ok {
					logrus.Fatal("non-PluginConfig type passed for uniquePlugins validator")
				}

				if !config.IsUnique() {
					continue
				}

				configs = append(configs, config)
			}
		}

		for _, el := range configs {
			for _, other := range configs {
				if el == other {
					continue
				}

				if reflect.TypeOf(el) == reflect.TypeOf(other) {
					return false
				}
			}
		}

		return true
	}); err != nil {
		logrus.Fatal("failed to register uniquePlugins validator")
	}
}
