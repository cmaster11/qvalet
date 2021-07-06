package main

import (
	"github.com/go-playground/validator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

/// [config-docs]
type Config struct {

	// HTTP port used by go-to-exec to listen for incoming requests
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`

	// Map of path -> listener
	Listeners map[string]*ListenerConfig `mapstructure:"listeners"`
}

type ListenerConfig struct {

	// Command to run
	Command string `mapstructure:"command" validate:"required"`

	// Arguments for `Command`
	Args []string `mapstructure:"args"`

	// If true, logs output
	LogOutput bool `mapstructure:"logOutput"`

	// If true, logs args
	LogArgs bool `mapstructure:"logArgs"`

	// If true, returns command execution output to request
	ReturnOutput bool `mapstructure:"returnOutput"`

	// Which methods to enable for this listener. Defaults to GET, POST
	// MUST be UPPERCASE!
	Methods []string `mapstructure:"methods"`
}

/// [config-docs]

var validate = validator.New()

func MustLoadConfig(filename string) *Config {
	// TODO: once Viper supports casing, replace
	// Ref: https://github.com/spf13/viper/pull/860
	myViper := viper.NewWithOptions(viper.KeyPreserveCase())

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
