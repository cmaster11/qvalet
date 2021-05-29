package main

import (
	"github.com/go-playground/validator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Listener struct {
	Command string `mapstructure:"command" validate:"required"`
}

type Config struct {
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`

	Listeners map[string]*Listener `mapstructure:"listeners"`
}

var validate = validator.New()

func MustLoadConfig(filename string) *Config {
	viper.SetEnvPrefix("GTE")
	viper.AutomaticEnv()

	if filename != "" {
		viper.SetConfigFile(filename)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	}

	err := viper.ReadInConfig()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to load config")
	}

	config := new(Config)

	if err := viper.Unmarshal(config); err != nil {
		logrus.WithError(err).Fatalf("failed to unmarshal config")
	}

	if err := validate.Struct(config); err != nil {
		logrus.WithError(err).Fatalf("failed to validate config")
	}

	return config
}
