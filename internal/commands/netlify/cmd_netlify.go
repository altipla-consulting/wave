package netlify

import (
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/internal/gerrit"
	"github.com/altipla-consulting/wave/internal/query"
)

type cmdFlags struct {
	Tag    string
	Source string
}

var (
	flags cmdFlags
)

func init() {
	Cmd.PersistentFlags().StringVar(&flags.Tag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	Cmd.PersistentFlags().StringVar(&flags.Source, "source", "", "Source folder. Defaults to the name of the application.")
}

var Cmd = &cobra.Command{
	Use:     "netlify",
	Short:   "Deploy a site to Netlify.",
	Example: "wave netlify foo",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		site := args[0]

		if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" && flags.Tag == "" {
			flags.Tag = gerrit.Descriptor()
		}

		log.Info("Get last commit message")
		cmd := exec.Command("git", "log", "-1", "--pretty=%B")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Trace(err)
		}
		var filtered []string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "Change-Id") {
				continue
			}
			filtered = append(filtered, line)
		}
		lastCommit := strings.TrimSpace(strings.Join(filtered, "\n"))

		log.WithFields(log.Fields{
			"name":    site,
			"version": query.Version(),
		}).Info("Deploy Netlify site")

		netlify := []string{
			"netlify",
			"deploy",
			"--dir", "dist",
			"--json",
			"--message", lastCommit,
		}
		if flags.Tag != "" {
			netlify = append(netlify, "--alias", flags.Tag)
		}
		if os.Getenv("BUILD_CAUSE") != "SCMTRIGGER" {
			netlify = append(netlify, "--prod")
		}
		log.Debug(strings.Join(netlify, " "))
		build := exec.Command(netlify[0], netlify[1:]...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Dir = flags.Source
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}
