package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdWorker = &cobra.Command{
	Use:     "worker",
	Short:   "Deploy a container to Cloud Run Worker Pools.",
	Example: "wave worker foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	const maxDeployAttempts = 2

	var flagProject, flagRegion, flagRepo string
	var flagSentry string
	cmdWorker.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdWorker.Flags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdWorker.Flags().StringVar(&flagRepo, "repo", "", "Artifact Registry repository name where the container is stored.")
	cmdWorker.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdWorker.MarkFlagRequired("sentry")

	cmdWorker.RunE = func(command *cobra.Command, args []string) error {
		if flagProject == "" {
			flagProject = env.GoogleProject()
		}

		client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}
		org := sentry.Organization{
			Slug: sentryAPIString("altipla"),
		}
		keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(flagSentry)})
		if err != nil {
			return errors.Trace(err)
		}

		version := query.Version(command.Context())

		worker := args[0]

		slog.Info("Deploy worker", slog.String("name", worker), slog.String("version", version))

		imageTag := query.VersionImageTag(command.Context())
		image := "eu.gcr.io/" + flagProject + "/" + worker + ":" + imageTag
		if flagRepo != "" {
			image = fmt.Sprintf("europe-west1-docker.pkg.dev/%s/%s/%s:%s", flagProject, flagRepo, worker, imageTag)
		}

		env := []string{
			"SENTRY_DSN=" + keys[0].DSN.Public,
			"VERSION=" + version,
		}
		gcloud := []string{
			"beta", "run", "worker-pools", "deploy",
			worker,
			"--image", image,
			"--region", flagRegion,
			"--platform", "managed",
			"--update-env-vars", strings.Join(env, ","),
			"--labels", "app=" + worker,
		}

		slog.Debug(strings.Join(append([]string{"gcloud"}, gcloud...), " "))

		for attempt := 0; attempt < maxDeployAttempts; attempt++ {
			build := exec.Command("gcloud", gcloud...)
			build.Stdout = os.Stdout
			var buf bytes.Buffer
			build.Stderr = io.MultiWriter(os.Stderr, &buf)
			if err = build.Run(); err != nil {
				if shouldRetryDeployWorker(buf.String()) {
					slog.Warn("Deployment failed because of a concurrent operation. Retrying in a moment.")
					time.Sleep(time.Duration(rand.Intn(15)+1) * time.Second)
					continue
				}
				return errors.Trace(err)
			}
			break
		}

		return nil
	}
}

func shouldRetryDeployWorker(s string) bool {
	if strings.Contains(s, "ABORTED: Conflict for resource") && strings.Contains(s, "was specified but current version is") {
		return true
	}
	if strings.Contains(s, "Resource readiness deadline exceeded") {
		return true
	}

	return false
}
