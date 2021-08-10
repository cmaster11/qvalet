package pkg

import (
	"github.com/go-playground/validator"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// @formatter:off
/// [config-docs]
type Config struct {
	// If true, enable debug logs
	Debug bool `mapstructure:"debug"`

	// HTTP port used by go-to-exec to listen for incoming requests
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`

	// Map of route -> listener
	Listeners map[string]*ListenerConfig `mapstructure:"listeners" validate:"-"`

	// Holds default configs valid for all listeners.
	// Values defined in each listener will overwrite these ones.
	Defaults ListenerConfig `mapstructure:"defaults" validate:"-"`

	// If specified, defines the HTTP username for all listeners' basic auth.
	// Defaults to "gte".
	HTTPAuthUsername string `mapstructure:"httpAuthUsername"`
}

type ListenerConfig struct {
	// Command to run
	Command string `mapstructure:"command" validate:"required"`

	// Arguments for `Command`
	Args []string `mapstructure:"args"`

	// Environment variables to pass to the command
	Env map[string]string `mapstructure:"env"`

	// Which methods to enable for this listener. Defaults to GET, POST
	// MUST be UPPERCASE!
	Methods []string `mapstructure:"methods"`

	// Define which temporary files you want to create
	Files map[string]string `mapstructure:"files"`

	// If populated, only requests with the right auth credentials will be
	// accepted for this listener.
	ApiKeys []string `mapstructure:"apiKeys"`

	// If true, logs output of Command
	LogOutput *bool `mapstructure:"logOutput"`

	// If true, logs args passed in the request
	LogArgs *bool `mapstructure:"logArgs"`

	// If true, logs the executed command with args
	LogCommand *bool `mapstructure:"logCommand"`

	// If true, returns Command execution output in the response
	ReturnOutput *bool `mapstructure:"returnOutput"`

	// If defined, triggers a command whenever an error is raised in
	// the execution of the current listener.
	ErrorHandler *ListenerConfig `mapstructure:"errorHandler" validate:"-"`
}

/// [config-docs]
// @formatter:on

var validate = validator.New()

func mergeListenerConfig(defaults *ListenerConfig, listenerConfig *ListenerConfig) (*ListenerConfig, error) {
	// Merge with the defaults
	mergedConfig := &ListenerConfig{}
	if err := mergo.Merge(mergedConfig, defaults, mergo.WithOverride, mergo.WithTransformers(mergoTransformerCustomInstance)); err != nil {
		return nil, errors.WithMessage(err, "failed to merge defaults config")
	}
	if err := mergo.Merge(mergedConfig, listenerConfig, mergo.WithOverride, mergo.WithTransformers(mergoTransformerCustomInstance)); err != nil {
		return nil, errors.WithMessage(err, "failed to merge overriding listener config")
	}
	return mergedConfig, nil
}

func MustLoadConfig(filename string) *Config {
	// TODO: once Viper supports casing, replace
	// Ref: https://github.com/spf13/viper/pull/860
	myViper := viper.NewWithOptions(
		viper.KeyPreserveCase(),
		// Lets us use . as file names in temporary files
		viper.KeyDelimiter("::"),
	)

	myViper.SetEnvPrefix("GTE")
	myViper.AutomaticEnv()

	if filename != "" {
		myViper.SetConfigFile(filename)
	} else {
		myViper.SetConfigName("config")
		myViper.SetConfigType("yaml")
		myViper.AddConfigPath(".")
		myViper.AddConfigPath("./config")
	}

	err := myViper.ReadInConfig()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to load config")
	}

	config := new(Config)

	if err := myViper.Unmarshal(config); err != nil {
		logrus.WithError(err).Fatalf("failed to unmarshal config")
	}

	if config.HTTPAuthUsername == "" {
		config.HTTPAuthUsername = "gte"
	}

	if err := validate.Struct(config); err != nil {
		logrus.WithError(err).Fatalf("failed to validate config")
	}

	return config
}
