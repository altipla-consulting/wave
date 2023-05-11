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

func init() {
	cmdRoot := cmdbase.CmdRoot(
		"wave",
		"Build and publish applications.",
		cmdbase.WithInstall(),
		cmdbase.WithUpdate("github.com/altipla-consulting/wave"))
	cmdRoot.AddCommand(cmdAR)
	cmdRoot.AddCommand(cmdBuild)
	cmdRoot.AddCommand(cmdDeploy)
	cmdRoot.AddCommand(cmdKubernetes)
	cmdRoot.AddCommand(cmdNetlify)
	cmdRoot.AddCommand(cmdPages)
	cmdRoot.AddCommand(cmdPreview)
}
