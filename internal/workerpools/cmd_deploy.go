package workerpools

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdDeploy = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy a container to Cloud Run Worker Pools.",
	Example: "wave worker-pools deploy foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagProject, flagRegion, flagRepo string
	var flagSentry string
	cmdDeploy.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdDeploy.Flags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdDeploy.Flags().StringVar(&flagRepo, "repo", "", "Artifact Registry repository name where the container is stored.")
	cmdDeploy.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdDeploy.MarkFlagRequired("sentry")

	cmdDeploy.RunE = func(command *cobra.Command, args []string) error {
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

		slog.Info("Deploy worker pools", slog.String("name", worker), slog.String("version", version))

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
			"run", "worker-pools", "deploy",
			worker,
			"--image", image,
			"--region", flagRegion,
			"--update-env-vars", strings.Join(env, ","),
		}

		slog.Debug(strings.Join(append([]string{"gcloud"}, gcloud...), " "))

		build := exec.Command("gcloud", gcloud...)
		build.Stdout = os.Stdout
		var buf bytes.Buffer
		build.Stderr = io.MultiWriter(os.Stderr, &buf)
		if err = build.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}

func sentryAPIString(s string) *string {
	return &s
}
