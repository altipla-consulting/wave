package pages

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

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
		}
		wrangler = append(wrangler, flagSource)
		log.Debug(strings.Join(wrangler, " "))
		publish := exec.Command(wrangler[0], wrangler[1:]...)
		var buf bytes.Buffer
		publish.Stdout = io.MultiWriter(os.Stdout, &buf)
		publish.Stderr = os.Stderr
		publish.Env = os.Environ()
		publish.Env = append(publish.Env, fmt.Sprintf("CLOUDFLARE_ACCOUNT_ID=%s", flagAccount))
		if err := publish.Run(); err != nil {
			return errors.Trace(err)
		}

		match, err := regexp.Compile("https://[^.]+\\." + flagProject + "\\.pages\\.dev")
		if err != nil {
			return errors.Trace(err)
		}
		result := match.FindString(buf.String())
		if result == "" {
			return errors.Errorf("cannot find preview URL in wrangler output")
		}

		if !gerrit.IsPreview() {
			if err := gerrit.Comment(fmt.Sprintf("Preview %s: %s", flagProject, result)); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	},
}
