package debug

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug commands.",
}

func init() {
	Cmd.AddCommand(cmdHost)
	Cmd.AddCommand(cmdVersion)
}
