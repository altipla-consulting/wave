package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/altipla-consulting/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	if err := cmdRoot.Execute(); err != nil {
		log.Error(err.Error())
		log.Debug(errors.Stack(err))
		os.Exit(1)
	}
}
