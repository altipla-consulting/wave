package containerapps

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "container-apps",
	Aliases: []string{"containerapps"},
	Short:   "Manage Azure Container Apps deployments",
}

func init() {
	Cmd.AddCommand(cmdDeploy)
	Cmd.AddCommand(cmdDeployJob)
	Cmd.AddCommand(cmdRunJob)
}
