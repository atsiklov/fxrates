package main

// @title FX Rates API
// @version 1.0
// @description API for scheduling and retrieving foreign exchange rates
// @BasePath /api/v1

import (
	"fxrates/internal/app"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := app.Run(); err != nil {
		logrus.WithError(err).Fatal("Application exited with error")
	}
}
