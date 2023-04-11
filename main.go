package main

import (
	"math/rand"
	"time"

	"github.com/altipla-consulting/cmdbase"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmdbase.Main()
}
