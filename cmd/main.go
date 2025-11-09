package main

import (
	"fxrates/internal/app"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := app.Run(); err != nil {
		logrus.WithError(err).Error("Application exited with error")
	}
}
