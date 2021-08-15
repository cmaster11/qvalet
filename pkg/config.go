package pkg

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
	// If true, enable all logging by default
	Debug bool `mapstructure:"debug"`

	// HTTP port used by go-to-exec to listen for incoming requests, defaults to 7055
	Port int `mapstructure:"port" validate:"min=0,max=65535"`

	// Map of route -> listener
	Listeners map[string]*ListenerConfig `mapstructure:"listeners" validate:"-"`

	// Holds default configs valid for all listeners in this config.
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

	// Storage configuration
	Storage *StorageConfig `mapstructure:"storage"`
}

type StorageConfig struct {
	// Connection string for the storage service where payloads will be stored.
	// Ref: https://beyondstorage.io/docs/go-storage/services/index
	Conn string `mapstructure:"conn" validate:"required"`

	// If true, persist every request's args
	StoreArgs bool `mapstructure:"storeArgs"`

	// If true, persist every request's executed command, its args and env vars
	StoreCommand bool `mapstructure:"storeCommand"`

	// If true, persist every request's executed command result
	StoreOutput bool `mapstructure:"storeOutput"`
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

var Validate = validator.New()

func init() {
	if err := Validate.RegisterValidation("authHeaderMethod", func(fl validator.FieldLevel) bool {
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

func MergeListenerConfig(defaults *ListenerConfig, listenerConfig *ListenerConfig) (*ListenerConfig, error) {
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

func MustLoadConfigs(filenames ...string) map[int][]*Config {
	var toMerge []*Config

	for _, filename := range filenames {
		subConfig, err := LoadConfig(filename)
		if err != nil {
			logrus.WithError(err).Fatalf("failed to load sub config")
		}
		toMerge = append(toMerge, subConfig)
	}

	configsByPort, err := groupConfigsByPort(toMerge...)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to merge configs")
	}

	return configsByPort
}

func groupConfigsByPort(configs ...*Config) (map[int][]*Config, error) {
	ret := make(map[int][]*Config)

	// We want to merge configs by port
	for _, config := range configs {
		ret[config.Port] = append(ret[config.Port], config)
	}

	return ret, nil
}

func LoadConfig(filename string) (*Config, error) {
	myViper := readConfigToViper(filename, "config")

	config := new(Config)

	if err := myViper.Unmarshal(config,
		// Lets us decode custom configuration types
		viper.DecodeHook(defaultDecodeHook),
	); err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal config")
	}

	// Apply defaults
	if config.Port == 0 {
		config.Port = 7055
	}

	if config.Debug {
		newDefaults, err := MergeListenerConfig(&ListenerConfig{
			LogOutput:    boolPtr(true),
			LogCommand:   boolPtr(true),
			LogArgs:      boolPtr(true),
			ReturnOutput: boolPtr(true),
		}, &config.Defaults)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to merge debug config with listener config")
		}
		config.Defaults = *newDefaults
	}

	return config, nil
}

func LoadDefaults(filename string) (*ListenerConfig, error) {
	myViper := readConfigToViper(filename, "defaults")

	config := new(ListenerConfig)

	if err := myViper.Unmarshal(config,
		// Lets us decode custom configuration types
		viper.DecodeHook(defaultDecodeHook),
	); err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal defaults config")
	}

	return config, nil
}

func readConfigToViper(filename string, defaultFilename string) *viper.Viper {
	// If we got a stdin config, store it in a tmp file and then use
	// the tmp file as source
	if filename == "-" {
		tmp, err := os.CreateTemp("", fmt.Sprintf("gte-%s-*.yaml", defaultFilename))
		if err != nil {
			logrus.WithError(err).Fatalf("failed to create temporary file")
		}

		content, err := readStdin()
		if err != nil {
			logrus.WithError(err).Fatalf("failed to read stdin for config")
		}

		if err := os.WriteFile(tmp.Name(), []byte(content), 0444); err != nil {
			logrus.WithError(err).Fatalf("failed to store stdin config to temporary file")
		}

		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.WithField("content", content).Debug("using config from stdin")
		} else {
			logrus.Info("using config from stdin")
		}

		filename = tmp.Name()
	}

	myViper := getViper(filename, defaultFilename)

	err := myViper.ReadInConfig()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to load config")
	}
	return myViper
}

func getViper(filename string, defaultName string) *viper.Viper {
	// TODO: once Viper supports casing, replace
	// Ref: https://github.com/spf13/viper/pull/860
	myViper := viper.NewWithOptions(
		// viper.KeyPreserveCase(),
		// Lets us use . as file names in temporary files
		viper.KeyDelimiter("::"),
	)

	myViper.SetEnvPrefix("GTE")
	myViper.SetEnvKeyReplacer(strings.NewReplacer("::", "_", "/", "_"))
	myViper.AutomaticEnv()

	if filename != "" {
		myViper.SetConfigFile(filename)
	} else {
		myViper.SetConfigName(defaultName)
		myViper.SetConfigType("yaml")
		myViper.AddConfigPath(".")
		myViper.AddConfigPath("./config")
	}
	return myViper
}

func readStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	txt := ""
	for scanner.Scan() {
		txt += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		return "", errors.WithMessage(err, "failed to read stdin")
	}
	return txt, nil
}
