package pkg

import (
	"github.com/go-playground/validator"
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
	Listeners map[string]*ListenerConfig `mapstructure:"listeners"`
}

type ListenerConfig struct {

	// Command to run
	Command string `mapstructure:"command" validate:"required"`

	// Arguments for `Command`
	Args []string `mapstructure:"args"`

	// If true, logs output of Command
	LogOutput bool `mapstructure:"logOutput"`

	// If true, logs args passed in the request
	LogArgs bool `mapstructure:"logArgs"`

	// If true, logs the executed command with args
	LogCommand bool `mapstructure:"logCommand"`

	// If true, returns Command execution output in the response
	ReturnOutput bool `mapstructure:"returnOutput"`

	// Which methods to enable for this listener. Defaults to GET, POST
	// MUST be UPPERCASE!
	Methods []string `mapstructure:"methods"`

	// Define which temporary files you want to create
	Files map[string]string `mapstructure:"files"`
}

/// [config-docs]
// @formatter:on

var validate = validator.New()

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

	if err := validate.Struct(config); err != nil {
		logrus.WithError(err).Fatalf("failed to validate config")
	}

	return config
}
