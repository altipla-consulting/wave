package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/gerrit"
)

var cmdPages = &cobra.Command{
	Use:     "pages",
	Short:   "Deploy a Cloudflare Pages project.",
	Example: "wave pages",
	Args:    cobra.NoArgs,
}

func init() {
	var flagAccount, flagProject, flagSource string
	cmdPages.Flags().StringVar(&flagAccount, "account", "", "Cloudflare account ID.")
	cmdPages.Flags().StringVar(&flagProject, "project", "", "Cloudflare Pages project where the files will be deployed to.")
	cmdPages.Flags().StringVar(&flagSource, "source", "dist", "Source folder. Defaults to a folder named dist.")
	cmdPages.MarkFlagRequired("account")
	cmdPages.MarkFlagRequired("project")

	cmdPages.RunE = func(command *cobra.Command, args []string) error {
		logger := slog.With(slog.String("branch", gerrit.SimulatedBranch()))
		logger.Info("Build app")

		logger.Info("Deploy to Cloudflare Pages")
		wrangler := []string{
			"wrangler",
			"pages", "deploy",
			"--project-name", flagProject,
			"--commit-dirty=true",
		}
		if gerrit.IsPreview() {
			wrangler = append(wrangler, "--branch", gerrit.SimulatedBranch())
		} else {
			wrangler = append(wrangler, "--branch", "main")
		}
		wrangler = append(wrangler, flagSource)
		slog.Debug(strings.Join(wrangler, " "))
		deploy := exec.Command(wrangler[0], wrangler[1:]...)
		deploy.Stdout = os.Stdout
		deploy.Stderr = os.Stderr
		deploy.Env = os.Environ()
		deploy.Env = append(deploy.Env, fmt.Sprintf("CLOUDFLARE_ACCOUNT_ID=%s", flagAccount))
		if err := deploy.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
