package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/query"
)

var cmdACR = &cobra.Command{
	Use:     "acr",
	Short:   "Build a container from a predefined folder structure deploying to Azure Container Registry.",
	Example: "wave acr foo --repo foo-acr",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagRepo, flagSource string
	cmdACR.Flags().StringVar(&flagRepo, "repo", "", "Azure Container Registry repository name where the container will be stored.")
	cmdACR.Flags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")
	cmdACR.MarkFlagRequired("repo")

	cmdACR.RunE = func(cmd *cobra.Command, args []string) error {
		app := args[0]

		token := os.Getenv("ACR_TOKEN")
		if token == "" {
			return errors.Errorf("Missing ACR_TOKEN environment variable. Assign it with whisper for increased security.")
		}

		version := query.VersionImageTag(cmd.Context())
		logger := slog.With(slog.String("name", app), slog.String("version", version))
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

		logger.Info("Log in to Azure Container Registry")
		login := exec.CommandContext(cmd.Context(), "docker", "login", flagRepo+".azurecr.io", "-u", flagRepo, "-p", token)
		login.Stdout = os.Stdout
		login.Stderr = os.Stderr
		if err := login.Run(); err != nil {
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
