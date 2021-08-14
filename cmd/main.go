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

	if os.Getenv("GTE_DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	config := pkg.MustLoadConfig(flagConfigFilename)

	if config.Port == 0 {
		config.Port = 7055
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
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
