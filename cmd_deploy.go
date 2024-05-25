package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdDeploy = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy a container to Cloud Run.",
	Example: "wave deploy foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	const maxDeployAttempts = 2

	var flagProject, flagRegion, flagRepo string
	var flagSentry string
	var flagTag string
	cmdDeploy.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdDeploy.Flags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdDeploy.Flags().StringVar(&flagRepo, "repo", "", "Artifact Registry repository name where the container is stored.")
	cmdDeploy.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdDeploy.Flags().StringVar(&flagTag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	cmdDeploy.MarkFlagRequired("sentry")

	cmdDeploy.RunE = func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = env.GoogleProject()
		}

		client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}

		org := sentry.Organization{
			Slug: sentryAPIString("altipla-consulting"),
		}
		keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(flagSentry)})
		if err != nil {
			return errors.Trace(err)
		}

		version := query.Version(command.Context())

		log.WithFields(log.Fields{
			"name":    app,
			"version": version,
		}).Info("Deploy app")

		imageTag := query.VersionImageTag(command.Context())
		image := "eu.gcr.io/" + flagProject + "/" + app + ":" + imageTag
		if flagRepo != "" {
			image = fmt.Sprintf("europe-west1-docker.pkg.dev/%s/%s/%s:%s", flagProject, flagRepo, app, imageTag)
		}

		env := []string{
			"SENTRY_DSN=" + keys[0].DSN.Public,
			"VERSION=" + version,
		}
		gcloud := []string{
			"beta", "run", "deploy",
			app,
			"--image", image,
			"--region", flagRegion,
			"--platform", "managed",
			"--update-env-vars", strings.Join(env, ","),
			"--labels", "app=" + app,
		}
		if tag := query.VersionHostname(flagTag); tag != "" {
			if !query.IsRelease() {
				gcloud = append(gcloud, "--no-traffic")
			}
			gcloud = append(gcloud, "--tag", tag)
		}

		log.Debug(strings.Join(append([]string{"gcloud"}, gcloud...), " "))

		for attempt := 0; attempt < maxDeployAttempts; attempt++ {
			build := exec.Command("gcloud", gcloud...)
			build.Stdout = os.Stdout
			var buf bytes.Buffer
			build.Stderr = io.MultiWriter(os.Stderr, &buf)
			if err = build.Run(); err != nil {
				if shouldRetryDeploy(buf.String()) {
					log.Warning("Deployment failed because of a concurrent operation. Retrying in a moment.")
					time.Sleep(time.Duration(rand.Intn(15)+1) * time.Second)
					continue
				}
				return errors.Trace(err)
			}
			break
		}

		if query.IsRelease() && flagTag == "" {
			log.WithFields(log.Fields{
				"name":    app,
				"version": version,
			}).Info("Enable traffic to the latest version of the app")

			traffic := exec.Command(
				"gcloud",
				"run", "services", "update-traffic",
				app,
				"--project", flagProject,
				"--region", flagRegion,
				"--to-latest",
			)
			traffic.Stdout = os.Stdout
			traffic.Stderr = os.Stderr
			if err := traffic.Run(); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}
}

func shouldRetryDeploy(s string) bool {
	if strings.Contains(s, "ABORTED: Conflict for resource") && strings.Contains(s, "was specified but current version is") {
		return true
	}
	if strings.Contains(s, "Resource readiness deadline exceeded") {
		return true
	}

	return false
}
