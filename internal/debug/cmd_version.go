package debug

import (
	"fmt"

	"github.com/altipla-consulting/wave/internal/query"
	"github.com/spf13/cobra"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Short:   "Debug the version that is taken from a project.",
	Example: "wave debug version example",
	Args:    cobra.ExactArgs(0),
}

func init() {
	cmdVersion.RunE = func(cmd *cobra.Command, args []string) error {
		version := query.Version()
		fmt.Println(version)
		return nil
	}
}
