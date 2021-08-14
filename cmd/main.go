package main

import (
	"fmt"
	"net/http"
	"os"

	"gotoexec/pkg"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var opts struct {
	ConfigFilename string `short:"c" long:"config" description:"Configuration file path"`
	DotEnvFilename string `short:"e" long:"dotenv" description:"dotenv file path"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to parse flags")
	}

	if opts.DotEnvFilename != "" {
		if err := godotenv.Load(opts.DotEnvFilename); err != nil {
			logrus.WithField("file", opts.DotEnvFilename).WithError(err).Fatal("failed to load .env file")
		}
	}

	// Internal debug logging
	if os.Getenv("GTE_VERBOSE") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	config := pkg.MustLoadConfig(opts.ConfigFilename)

	// Unless there is a particular reason, gin should always be in release mode
	if os.Getenv(gin.EnvGinMode) != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(gin.ErrorLogger())

	router.GET("/healthz", func(context *gin.Context) {
		context.AbortWithStatus(http.StatusOK)
	})

	gte := pkg.NewGoToExec(config)
	gte.MountRoutes(router)

	logrus.WithField("port", config.Port).Info("server listening")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router); err != nil {
		logrus.WithError(err).Fatalf("failed to start server")
	}
}
