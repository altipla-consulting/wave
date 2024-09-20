package main

import (
	"github.com/altipla-consulting/cmdbase"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"

	"github.com/altipla-consulting/wave/internal/debug"
)

func main() {
	cmdbase.Main()
}

func init() {
	godotenv.Load(".env.production", ".env")

	cmdRoot := cmdbase.CmdRoot(
		"wave",
		"Build and publish applications.",
		cmdbase.WithInstall(),
		cmdbase.WithUpdate("github.com/altipla-consulting/wave"))
	cmdRoot.AddCommand(cmdACR)
	cmdRoot.AddCommand(cmdAR)
	cmdRoot.AddCommand(cmdBuild)
	cmdRoot.AddCommand(cmdCompose)
	cmdRoot.AddCommand(cmdDeploy)
	cmdRoot.AddCommand(cmdECR)
	cmdRoot.AddCommand(cmdJob)
	cmdRoot.AddCommand(cmdKubernetes)
	cmdRoot.AddCommand(cmdLightsail)
	cmdRoot.AddCommand(cmdNetlify)
	cmdRoot.AddCommand(cmdPages)
	cmdRoot.AddCommand(cmdPreview)
	cmdRoot.AddCommand(cmdVersion)
	cmdRoot.AddCommand(debug.Cmd)
}
