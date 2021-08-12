package pkg

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
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

	// If defined, the hook will be triggered only if this condition is met
	Trigger *IfTemplate `mapstructure:"trigger"`

	// List of allowed authentication methods
	Auth []*AuthConfig `mapstructure:"auth" validate:"dive"`

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
/// [auth-docs]

type AuthConfig struct {
	// Api keys for this auth type
	ApiKeys []string `mapstructure:"apiKeys" validate:"required"`

	// If true, allows basic HTTP authentication
	BasicAuth bool `mapstructure:"basicAuth"`

	// If true, url query authentication will be allowed
	QueryAuth bool `mapstructure:"queryAuth"`

	// The key to check for in the url query.
	// Defaults to __gteApiKey if none is provided
	QueryAuthKey string `mapstructure:"queryAuthKey"`

	// The basic auth HTTP username.
	// Defaults to `gte` if none is provided
	BasicAuthUser string `mapstructure:"basicAuthUser"`

	// If provided, apiKeys will be searched for in these headers
	// E.g. GitLab hooks can authenticate via X-Gitlab-Token
	AuthHeaders []*AuthHeader `mapstructure:"authHeaders" validate:"dive"`
}

type AuthHeader struct {
	// Header name, case-insensitive
	Header string `mapstructure:"header"`

	// If provided, the header content will be compared using this method
	Method AuthHeaderMethod `mapstructure:"method" validate:"authHeaderMethod"`

	// If provided, this is used to alter the incoming header value, where
	// the header value is the current context `.`
	// E.g. for GitHub webhooks, `{{ replace "sha256=" "" . }}` would strip out the
	// initial sha256= prefix GitHub passes to all webhooks
	Transform *Template `mapstructure:"transform"`
}

type AuthHeaderMethod string

const (
	// Simply compares the value of the header with every api key
	AuthHeaderMethodNone AuthHeaderMethod = ""

	// Calculates the payload HMAC-SHA256 hash for each api key,
	// and compares the hash with the value provided in the header.
	AuthHeaderMethodHMACSHA256 AuthHeaderMethod = "hmac-sha256"
)

/// [auth-docs]
// @formatter:on

var validate = validator.New()

func init() {
	if err := validate.RegisterValidation("authHeaderMethod", func(fl validator.FieldLevel) bool {
		method := AuthHeaderMethod(fl.Field().String())
		switch method {
		case AuthHeaderMethodNone:
			return true
		case AuthHeaderMethodHMACSHA256:
			return true
		default:
			return false
		}
	}); err != nil {
		logrus.Fatal("failed to register authHeaderMethod validator")
	}
}

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

var defaultDecodeHook = mapstructure.ComposeDecodeHookFunc(
	// Default
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),

	// Custom
	StringToPointerIfTemplateHookFunc(),
	StringToPointerTemplateHookFunc(),
)

func init() {
	// Remove unnecessary logging
	spew.Config.DisablePointerAddresses = true
	spew.Config.DisableCapacities = true
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

	if err := myViper.Unmarshal(config,
		// Lets us decode custom configuration types
		viper.DecodeHook(defaultDecodeHook),
	); err != nil {
		logrus.WithError(err).Fatalf("failed to unmarshal config")
	}

	if err := validate.Struct(config); err != nil {
		logrus.WithError(err).Fatalf("failed to validate config")
	}

	return config
}
