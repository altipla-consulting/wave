package debug

import (
	"fmt"

	"github.com/altipla-consulting/wave/internal/query"
	"github.com/spf13/cobra"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Short:   "Debug the deployment version.",
	Example: "wave debug version",
	Args:    cobra.ExactArgs(0),
}

func init() {
	cmdVersion.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Println(query.Version())
		return nil
	}
}
