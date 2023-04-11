package main

import (
	"github.com/altipla-consulting/cmdbase"
	"github.com/altipla-consulting/wave/internal/commands/ar"
	"github.com/altipla-consulting/wave/internal/commands/build"
	"github.com/altipla-consulting/wave/internal/commands/compose"
	"github.com/altipla-consulting/wave/internal/commands/deploy"
	"github.com/altipla-consulting/wave/internal/commands/kubernetes"
	"github.com/altipla-consulting/wave/internal/commands/netlify"
	"github.com/altipla-consulting/wave/internal/commands/pages"
	"github.com/altipla-consulting/wave/internal/commands/preview"
)

func init() {
	cmdRoot := cmdbase.CmdRoot("wave", "Build and publish applications.")
	cmdRoot.AddCommand(ar.Cmd)
	cmdRoot.AddCommand(build.Cmd)
	cmdRoot.AddCommand(compose.Cmd)
	cmdRoot.AddCommand(deploy.Cmd)
	cmdRoot.AddCommand(kubernetes.Cmd)
	cmdRoot.AddCommand(netlify.Cmd)
	cmdRoot.AddCommand(pages.Cmd)
	cmdRoot.AddCommand(preview.Cmd)
}
