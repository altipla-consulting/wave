package main

import (
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"libs.altipla.consulting/errors"
)

func main() {
	if err := cmdRoot.Execute(); err != nil {
		rand.Seed(time.Now().UnixNano())
		log.Error(err.Error())
		log.Debug(errors.Stack(err))
		os.Exit(1)
	}
}
