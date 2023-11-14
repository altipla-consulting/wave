package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/altipla-consulting/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/query"
)

var cmdECR = &cobra.Command{
	Use:     "ecr",
	Short:   "Build a container from a predefined folder structure deploying to AWS ECR.",
	Example: "wave ecr foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagRepo, flagSource, flagRegion string
	cmdECR.Flags().StringVar(&flagRepo, "repo", "", "ECR repository name where the container will be stored.")
	cmdECR.Flags().StringVar(&flagSource, "source", "", "Source folder. Defaults to a folder with the name of the app.")
	cmdECR.Flags().StringVar(&flagRegion, "region", "eu-west-1", "AWS region where the container will be stored.")
	cmdECR.MarkFlagRequired("repo")

	cmdECR.RunE = func(command *cobra.Command, args []string) error {
		app := args[0]

		version := query.Version(command.Context())
		logger := log.WithFields(log.Fields{
			"name":    app,
			"version": version,
		})
		logger.Info("Build app")

		source := app
		if flagSource != "" {
			source = flagSource
		}
		image := fmt.Sprintf("%s/%s", flagRepo, app)

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

		build := exec.Command("docker", docker...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Log in to ECR")
		login := exec.Command("aws", "ecr", "get-login-password", "--region", flagRegion)
		var buf bytes.Buffer
		login.Stdout = &buf
		login.Stderr = os.Stderr
		if err := login.Run(); err != nil {
			return errors.Trace(err)
		}

		login = exec.Command("docker", "login", "--username", "AWS", "--password-stdin", flagRepo)
		login.Stdin = &buf
		login.Stdout = os.Stdout
		login.Stderr = os.Stderr
		if err := login.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Push to ECR")
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
	}
}
