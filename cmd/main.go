package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"gotoexec/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	flagConfigFilename string
)

func main() {
	flag.StringVar(&flagConfigFilename, "config", "", "Configuration file name")
	flag.StringVar(&flagConfigFilename, "c", "", "Configuration file name (shorthand)")

	flag.Parse()

	// Internal debug logging
	if os.Getenv("GTE_VERBOSE") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	config := pkg.MustLoadConfig(flagConfigFilename)

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
