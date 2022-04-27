package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"libs.altipla.consulting/errors"
)

func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Error(err.Error())
		log.Debug(errors.Stack(err))
		os.Exit(1)
	}
}
