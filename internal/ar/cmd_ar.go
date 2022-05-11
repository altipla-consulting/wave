package ar

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"
)

var (
	flagProject string
	flagRepo    string
	flagSource  string
)

func init() {
	Cmd.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	Cmd.Flags().StringVar(&flagRepo, "repo", "", "Artifact Registry repository name where the container will be stored.")
	Cmd.Flags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")
	Cmd.MarkFlagRequired("repo")
}

var Cmd = &cobra.Command{
	Use:     "ar",
	Short:   "Build a container from a predefined folder structure deploying to Artifact Registry.",
	Example: "wave ar foo",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = os.Getenv("GOOGLE_PROJECT")
		}

		version := time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
		if ref := os.Getenv("GITHUB_REF"); ref != "" {
			version = path.Base(ref)
		}
		if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
			version += ".preview"
		}

		logger := log.WithFields(log.Fields{
			"name":    app,
			"version": version,
		})
		logger.Info("Build app")

		source := app
		if flagSource != "" {
			source = flagSource
		}
		image := fmt.Sprintf("europe-west1-docker.pkg.dev/%s/%s/%s", flagProject, flagRepo, app)
		build := exec.Command(
			"docker",
			"build",
			"--cache-from", image+":latest",
			"-f", source+"/Dockerfile",
			"-t", image+":latest",
			"-t", image+":"+version,
			".",
		)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to Artifact Registry")
		push := exec.Command("docker", "push", image+":latest")
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		push = exec.Command("docker", "push", image+":"+version)
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}
