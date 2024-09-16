package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/query"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Short:   "Debug the deployment version.",
	Example: "wave debug version",
	Args:    cobra.ExactArgs(0),
}

func init() {
	var flagImageTag bool
	cmdVersion.Flags().BoolVar(&flagImageTag, "image-tag", false, "Print the image tag.")

	cmdVersion.RunE = func(cmd *cobra.Command, args []string) error {
		if flagImageTag {
			fmt.Println(query.VersionImageTag(cmd.Context()))
			return nil
		}

		fmt.Println(query.Version(cmd.Context()))
		return nil
	}
}
