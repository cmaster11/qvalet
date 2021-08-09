package main

import (
	"flag"
	"fmt"
	"net/http"

	"gotoexec/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	flagConfigFilename = flag.String("config", "", "Configuration file name")
)

func main() {
	flag.Parse()

	config := pkg.MustLoadConfig(*flagConfigFilename)

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	router := gin.Default()
	router.Use(gin.ErrorLogger())

	router.GET("/healthz", func(context *gin.Context) {
		context.AbortWithStatus(http.StatusOK)
	})

	gte := pkg.NewGoToExec(config)
	gte.MountRoutes(router)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router); err != nil {
		logrus.WithError(err).Fatalf("failed to start server")
	}
}
