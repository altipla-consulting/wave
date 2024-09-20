package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/altipla-consulting/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/query"
)

var cmdACR = &cobra.Command{
	Use:     "acr",
	Short:   "Build a container from a predefined folder structure deploying to Azure Container Registry.",
	Example: "wave ar foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagRepo, flagSource string
	cmdACR.Flags().StringVar(&flagRepo, "repo", "", "Azure Container Registry repository name where the container will be stored.")
	cmdACR.Flags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")
	cmdACR.MarkFlagRequired("repo")

	cmdACR.RunE = func(cmd *cobra.Command, args []string) error {
		app := args[0]

		version := query.VersionImageTag(cmd.Context())
		logger := log.WithFields(log.Fields{
			"name":    app,
			"version": version,
		})
		logger.Info("Build app")

		source := app
		if flagSource != "" {
			source = flagSource
		}
		image := fmt.Sprintf("%s.azurecr.io/%s", flagRepo, app)

		docker := []string{
			"build",
			"--cache-from", image + ":latest",
			"-f", source + "/Dockerfile",
			"-t", image + ":latest",
			"-t", image + ":" + version,
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := os.Stat(filepath.Join(home, ".npmrc")); err != nil && !os.IsNotExist(err) {
		} else if err == nil {
			docker = append(docker, "--secret", "id=npmrc,src="+filepath.Join(home, ".npmrc"))
		}

		docker = append(docker, ".") // build context

		build := exec.CommandContext(cmd.Context(), "docker", docker...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to Azure Container Registry")
		push := exec.CommandContext(cmd.Context(), "docker", "push", image+":latest")
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		push = exec.CommandContext(cmd.Context(), "docker", "push", image+":"+version)
		push.Stdout = os.Stdout
		push.Stderr = os.Stderr
		if err := push.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
