package pages

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/gerrit"
)

var (
	flagAccount string
	flagProject string
	flagSource  string
)

func init() {
	Cmd.Flags().StringVar(&flagAccount, "account", "", "Cloudflare account ID.")
	Cmd.Flags().StringVar(&flagProject, "project", "", "Cloudflare Pages project where the files will be deployed to.")
	Cmd.Flags().StringVar(&flagSource, "source", "dist", "Source folder. Defaults to a folder named dist.")
	Cmd.MarkFlagRequired("account")
	Cmd.MarkFlagRequired("project")
}

var Cmd = &cobra.Command{
	Use:     "pages",
	Short:   "Deploy a Cloudflare Pages project.",
	Example: "wave pages",
	Args:    cobra.NoArgs,
	RunE: func(command *cobra.Command, args []string) error {
		logger := log.WithFields(log.Fields{
			"branch": gerrit.Descriptor(),
		})
		logger.Info("Build app")

		logger.Info("Publish to Cloudflare Pages")
		wrangler := []string{
			"wrangler",
			"pages", "publish",
			"--project-name", flagProject,
		}
		if gerrit.IsPreview() {
			wrangler = append(wrangler, "--branch", gerrit.Descriptor())
		} else {
			wrangler = append(wrangler, "--branch", "main")
		}
		wrangler = append(wrangler, flagSource)
		log.Debug(strings.Join(wrangler, " "))
		publish := exec.Command(wrangler[0], wrangler[1:]...)
		publish.Stdout = os.Stdout
		publish.Stderr = os.Stderr
		publish.Env = os.Environ()
		publish.Env = append(publish.Env, fmt.Sprintf("CLOUDFLARE_ACCOUNT_ID=%s", flagAccount))
		if err := publish.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}
