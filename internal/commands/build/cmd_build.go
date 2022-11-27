package build

import (
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/internal/query"
)

var (
	flagProject, flagSource, flagRepo string
)

func init() {
	Cmd.PersistentFlags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	Cmd.PersistentFlags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")
	Cmd.PersistentFlags().StringVar(&flagRepo, "repo", "eu.gcr.io", "Docker repository. Defaults to Container Registry.")
}

var Cmd = &cobra.Command{
	Use:     "build",
	Short:   "Build a container from a predefined folder structure.",
	Example: "wave build foo",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = os.Getenv("GOOGLE_PROJECT")
		}

		logger := log.WithFields(log.Fields{
			"name":    app,
			"version": query.Version(),
		})

		source := app
		if flagSource != "" {
			source = flagSource
		}

		logger.Info("Build app")
		build := exec.Command(
			"docker",
			"build",
			"--cache-from", image(app, "latest"),
			"-f", source+"/Dockerfile",
			"-t", image(app, query.Version()),
			"-t", image(app, "latest"),
			".",
		)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to Container Registry")
		push := exec.Command("docker", "push", image(app, query.Version()))
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		push = exec.Command("docker", "push", image(app, "latest"))
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}

func image(app, version string) string {
	return flagRepo + "/" + flagProject + "/" + app + ":" + version
}
