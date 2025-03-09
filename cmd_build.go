package main

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdBuild = &cobra.Command{
	Use:     "build",
	Short:   "Build a container from a predefined folder structure.",
	Example: "wave build foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagProject, flagSource string
	cmdBuild.PersistentFlags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdBuild.PersistentFlags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")

	cmdBuild.RunE = func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = env.GoogleProject()
		}

		logger := slog.With(slog.String("name", app), slog.String("version", query.Version(command.Context())))

		source := app
		if flagSource != "" {
			source = flagSource
		}

		imageTag := query.VersionImageTag(command.Context())

		logger.Info("Build app")
		buildArgs := []string{
			"build",
			"--cache-from", "eu.gcr.io/" + flagProject + "/" + app + ":latest",
			"-f", source + "/Dockerfile",
			"-t", "eu.gcr.io/" + flagProject + "/" + app + ":latest",
			"-t", "eu.gcr.io/" + flagProject + "/" + app + ":" + imageTag,
		}

		npmrc := os.Getenv("NPM_CONFIG_USERCONFIG")
		if npmrc == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return errors.Trace(err)
			}
			if _, err := os.Stat(filepath.Join(home, ".npmrc")); err != nil && !os.IsNotExist(err) {
				return errors.Trace(err)
			} else if err == nil {
				npmrc = filepath.Join(home, ".npmrc")
			}
		}
		if npmrc != "" {
			buildArgs = append(buildArgs, "--secret", "id=npmrc,src="+npmrc)
		}

		buildArgs = append(buildArgs, ".")
		build := exec.Command("docker", buildArgs...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to Container Registry")
		push := exec.Command("docker", "push", "eu.gcr.io/"+flagProject+"/"+app+":latest")
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		push = exec.Command("docker", "push", "eu.gcr.io/"+flagProject+"/"+app+":"+imageTag)
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
