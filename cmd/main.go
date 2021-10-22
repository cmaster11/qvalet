package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"gotoexec/pkg"
	"gotoexec/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var opts struct {
	ConfigFilenames  []string `short:"c" long:"config" description:"Configuration file path"`
	DotEnvFilenames  []string `short:"e" long:"dotenv" description:"dotenv file path"`
	DefaultsFilename string   `short:"f" long:"defaults" description:"Defaults configuration file path"`
	Debug            bool     `short:"d" long:"debug" description:"Enable the debug flag on all configs by default"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to parse flags")
	}

	for _, filename := range opts.DotEnvFilenames {
		if err := godotenv.Load(filename); err != nil {
			logrus.WithField("file", opts.DotEnvFilenames).WithError(err).Fatal("failed to load .env file")
		}
	}

	// Internal debug logging
	if os.Getenv("GTE_VERBOSE") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithField("env", os.Environ()).Debug("env")
	}

	var defaults *pkg.ListenerConfig
	if opts.DefaultsFilename != "" {
		_defaults, err := pkg.LoadDefaults(opts.DefaultsFilename)
		if err != nil {
			logrus.WithField("file", opts.DefaultsFilename).WithError(err).Fatal("failed to load defaults from file")
		}
		defaults = _defaults
	}

	configsByPort := pkg.MustLoadConfigs(opts.ConfigFilenames...)

	// Unless there is a particular reason, gin should always be in release mode
	if os.Getenv(gin.EnvGinMode) != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	var mountResults []*pkg.MountRoutesResult

	wg := sync.WaitGroup{}
	for port, configs := range configsByPort {
		port := port

		router := gin.Default()
		router.Use(gin.ErrorLogger())

		router.GET("/healthz", func(context *gin.Context) {
			context.AbortWithStatus(http.StatusOK)
		})

		for _, config := range configs {
			if opts.Debug {
				config.Debug = true
			}

			if defaults != nil {
				newDefaults, err := pkg.MergeListenerConfig(defaults, &config.Defaults)
				if err != nil {
					logrus.WithError(err).Fatalf("failed to merge defaults config")
				}
				config.Defaults = *newDefaults
			}

			if err := utils.Validate.Struct(config); err != nil {
				logrus.WithError(err).Fatalf("failed to validate config")
			}

			mountResult, err := pkg.MountRoutes(router, config, fmt.Sprintf("%d_", port))
			if err != nil {
				logrus.WithError(err).Fatalf("failed to mount routes")
			}

			mountResults = append(mountResults, mountResult)
		}

		for _, r := range mountResults {
			if err := r.PluginsStart(); err != nil {
				logrus.WithError(err).Fatalf("failed to start plugins")
			}
		}

		logrus.WithField("port", port).Info("server listening")
		wg.Add(1)
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
				logrus.WithError(err).Fatalf("failed to start server")
			}
			wg.Done()
		}()
	}
	defer func() {
		for _, r := range mountResults {
			r.PluginsStop()
		}
	}()
	defer pkg.CloseAllDBConnections()

	wg.Wait()
}
