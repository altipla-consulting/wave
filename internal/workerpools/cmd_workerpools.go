package workerpools

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "worker-pools",
	Aliases: []string{"workerpools"},
	Short:   "Manage Cloud Run Worker Pools deployments",
}

func init() {
	Cmd.AddCommand(cmdDeploy)
}
