package pkg

import (
	"fmt"
	"reflect"
	"time"

	"qvalet/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun/migrate"
)

var pluginCounter int

func GetNextPluginIdx() int {
	idx := pluginCounter
	pluginCounter++
	return idx
}

type PluginInterface interface {
	Clone(newListener *CompiledListener) (PluginInterface, error)
	Id() string
}

type PluginBase struct {
	id string
}

func (p *PluginBase) Id() string {
	return p.id
}

func NewPluginBase(idPrefix string) PluginBase {
	return PluginBase{id: fmt.Sprintf("plugin-%s-%d", idPrefix, GetNextPluginIdx())}
}

type PluginLifecycle interface {
	PluginInterface

	// Invoked after all routes are mounted and plugins can start operating
	OnStart() error

	// Invoked on server shutdown
	OnStop()
}

type PluginConfigValidateCheckOtherPlugins interface {
	PluginInterface

	ValidateCheckOtherPlugins(otherPlugins []PluginInterface) error
}

type PluginConfigNeedsDb interface {
	PluginInterface

	// If true, it marks that this plugin will require a database connection to work
	NeedsDb() bool

	// Returns the database initialization migrations
	Migrations() *migrate.Migrations
}

type PluginHookMountRoutes interface {
	PluginInterface

	// Called on initialization, allows plugins to mount additional routes
	HookMountRoutes(engine *gin.Engine)
}

type PluginHookGetMiddlewares interface {
	PluginInterface

	// Called on initialization, allows plugins to mount additional middlewares, BEFORE a route is mounted
	HookGetMiddlewares(method string) []gin.HandlerFunc
}

type PluginHookPreExecute interface {
	PluginInterface

	// Called at runtime, before listener execution, allows alteration of args
	HookPreExecute(args map[string]interface{}) (map[string]interface{}, error)
}

type PluginHookPostExecute interface {
	PluginInterface

	// Called at runtime, before listener execution, allows alteration of args
	HookPostExecute(commandResult *ExecCommandResult) error
}

type PluginHookOutput interface {
	PluginInterface

	// Called at runtime, after listener successful execution, allows alteration of output
	HookOutput(
		// NOTE: this context can be artificial at times (e.g. if during a delayed execution), which means
		// you CANNOT read from this context, but only write to it
		writeOnlyContext *gin.Context,
		args map[string]interface{},
		listenerResponse *ListenerResponse,
	) (handled bool, err error)
}

type HookShouldRetryInfo struct {
	// How much time has passed since the beginning of the execution of the command?
	Elapsed time.Duration

	// How long until the next retry?
	Delay time.Duration

	// Which retry are we at? Starts from 1
	RetryCount int
}

type PluginHookRetry interface {
	PluginInterface

	HookShouldRetry(currentHookRetryInfo *HookShouldRetryInfo, args map[string]interface{}, commandResult *ExecCommandResult) (*time.Duration, map[string]interface{}, error)
}

type PluginConfig interface {
	// If true, a plugin cannot be declared more than once for a listener
	IsUnique() bool

	// Instantiates the plugin related to this config
	NewPlugin(listener *CompiledListener) (PluginInterface, error)
}

type PluginEntryConfig struct {

	// AWS SNS plugin, to auto-confirm AWS SNS subscriptions and handle SNS notifications
	AWSSNS *PluginAWSSNSConfig `mapstructure:"awsSNS"`

	// HTTP response plugin, to alter HTTP response headers, status code, etc...
	HTTPResponse *PluginHTTPResponseConfig `mapstructure:"httpResponse"`

	// Preview plugin, used to preview the command which will be executed
	Preview *PluginPreviewConfig `mapstructure:"preview"`

	// You can use the Retry plugin to retry a command execution, depending on
	// any condition you want
	Retry *PluginRetryConfig `mapstructure:"retry"`

	// You can use the Schedule plugin to defer command executions in the future
	Schedule *PluginScheduleConfig `mapstructure:"schedule"`

	// ---

	// Debug plugin, for testing
	Debug *PluginDebugConfig `mapstructure:"debug"`
}

func (config *PluginEntryConfig) ToPluginList(listener *CompiledListener) ([]PluginInterface, error) {
	var plugins []PluginInterface

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
