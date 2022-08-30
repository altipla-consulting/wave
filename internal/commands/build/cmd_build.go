package build

import (
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/internal/query"
)

type cmdFlags struct {
	Project string
	Source  string
}

var (
	flags cmdFlags
)

func init() {
	Cmd.PersistentFlags().StringVar(&flags.Project, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	Cmd.PersistentFlags().StringVar(&flags.Source, "source", "", "Source folder. Defaults to a folder with the name of the app.")
}

var Cmd = &cobra.Command{
	Use:     "build",
	Short:   "Build a container from a predefined folder structure.",
	Example: "wave build foo",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		app := args[0]

		if flags.Project == "" {
			flags.Project = os.Getenv("GOOGLE_PROJECT")
		}

		logger := log.WithFields(log.Fields{
			"name":    app,
			"version": query.Version(),
		})

		source := app
		if flags.Source != "" {
			source = flags.Source
		}

		logger.Info("Build app")
		build := exec.Command(
			"docker",
			"build",
			"--cache-from", "eu.gcr.io/"+flags.Project+"/"+app+":latest",
			"-f", source+"/Dockerfile",
			"-t", "eu.gcr.io/"+flags.Project+"/"+app+":latest",
			"-t", "eu.gcr.io/"+flags.Project+"/"+app+":"+query.Version(),
			".",
		)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to Container Registry")
		push := exec.Command("docker", "push", "eu.gcr.io/"+flags.Project+"/"+app+":latest")
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		push = exec.Command("docker", "push", "eu.gcr.io/"+flags.Project+"/"+app+":"+query.Version())
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}
