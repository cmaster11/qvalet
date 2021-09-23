package plugins

import (
	"reflect"

	"gotoexec/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Plugin interface {
	// Called on initialization, allows plugins to mount additional routes
	MountRoutes(engine *gin.Engine, listenerRoute string, listenerHandler func(args map[string]interface{}) (interface{}, error))

	// Called at runtime, before listener execution, allows alteration of args
	HookPreExecute(args map[string]interface{}) (map[string]interface{}, error)
}

type PluginConfig interface {
	// If true, a plugin cannot be declared more than one time for a listener
	IsUnique() bool

	// Instantiates the plugin related to this config
	NewPlugin() (Plugin, error)
}

type PluginEntryConfig struct {

	// AWS SNS plugin, to auto-confirm AWS SNS subscriptions and handle SNS notifications
	AWSSNS *PluginAWSSNSConfig `mapstructure:"awsSNS"`

	// Debug plugin, for testing
	Debug *PluginDebugConfig `mapstructure:"debug"`
}

func (config *PluginEntryConfig) ToPluginList() ([]Plugin, error) {
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

		plugin, err := configField.NewPlugin()
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
