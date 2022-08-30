package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/commands/ar"
	"github.com/altipla-consulting/wave/internal/commands/build"
	"github.com/altipla-consulting/wave/internal/commands/compose"
	"github.com/altipla-consulting/wave/internal/commands/deploy"
	"github.com/altipla-consulting/wave/internal/commands/kubernetes"
	"github.com/altipla-consulting/wave/internal/commands/netlify"
	"github.com/altipla-consulting/wave/internal/commands/pages"
	"github.com/altipla-consulting/wave/internal/commands/preview"
)

var flagDebug bool

func init() {
	cmdRoot.PersistentFlags().BoolVarP(&flagDebug, "debug", "d", false, "Enable debug logging for this tool.")
	cmdRoot.AddCommand(ar.Cmd)
	cmdRoot.AddCommand(build.Cmd)
	cmdRoot.AddCommand(compose.Cmd)
	cmdRoot.AddCommand(deploy.Cmd)
	cmdRoot.AddCommand(kubernetes.Cmd)
	cmdRoot.AddCommand(netlify.Cmd)
	cmdRoot.AddCommand(pages.Cmd)
	cmdRoot.AddCommand(preview.Cmd)
}

var cmdRoot = &cobra.Command{
	Use:          "wave",
	Short:        "Build and publish applications.",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetFormatter(new(log.TextFormatter))
		if flagDebug {
			log.SetLevel(log.DebugLevel)
		}
	},
}
