package debug

import (
	"fmt"

	"github.com/altipla-consulting/wave/internal/query"
	"github.com/spf13/cobra"
)

var cmdHash = &cobra.Command{
	Use:     "hash",
	Short:   "Debug the short hash from a git project.",
	Example: "wave debug hash example",
	Args:    cobra.ExactArgs(0),
}

func init() {
	cmdHash.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		hash := query.LastHash(ctx)
		fmt.Println(hash)
		return nil
	}
}
