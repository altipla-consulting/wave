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

var cmdJob = &cobra.Command{
	Use:     "job",
	Short:   "Deploy a container to Cloud Run Jobs.",
	Example: "wave job foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	const maxDeployAttempts = 2

	var flagProject, flagRegion, flagRepo string
	var flagMemory, flagServiceAccount, flagSentry string
	var flagEnv, flagEnvSecret, flagCloudSQL []string
	cmdJob.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdJob.Flags().StringVar(&flagMemory, "memory", "", "Memory available inside the Cloud Run application. Default: 256Mi.")
	cmdJob.Flags().StringVar(&flagServiceAccount, "service-account", "", "Service account. Defaults to one with the name of the application.")
	cmdJob.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdJob.Flags().StringSliceVar(&flagEnvSecret, "env-secret", nil, "Secrets to mount as environment variables.")
	cmdJob.Flags().StringSliceVar(&flagEnv, "env", nil, "Custom environment variables to define as `KEY=value` pairs.")
	cmdJob.Flags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdJob.Flags().StringSliceVar(&flagCloudSQL, "cloudsql", nil, "CloudSQL instances to connect to. Only the name.")
	cmdAR.Flags().StringVar(&flagRepo, "repo", "", "Artifact Registry repository name where the container is stored.")
	cmdJob.MarkFlagRequired("sentry")
	cmdJob.MarkFlagRequired("repo")

	cmdJob.RunE = func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = env.GoogleProject()
		}
		if flagServiceAccount == "" {
			flagServiceAccount = app
		}
		if flagMemory == "" {
			flagMemory = "512Mi"
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

		version := query.Version()

		log.WithFields(log.Fields{
			"name":            app,
			"version":         version,
			"memory":          flagMemory,
			"service-account": flagServiceAccount,
		}).Info("Deploy app")

		env := []string{
			"SENTRY_DSN=" + keys[0].DSN.Public,
		}
		env = append(env, flagEnv...)

		gcloud := []string{
			"beta", "run", "jobs", "deploy",
			app,
			"--image", fmt.Sprintf("europe-west1-docker.pkg.dev/%s/%s/%s:%s", flagProject, flagRepo, app, version),
			"--region", flagRegion,
			"--task-timeout", "10m",
			"--service-account", flagServiceAccount + "@" + flagProject + ".iam.gserviceaccount.com",
			"--memory", flagMemory,
			"--set-env-vars", strings.Join(env, ","),
			"--labels", "app=" + app,
		}
		if len(flagEnvSecret) > 0 {
			var secrets []string
			for _, secret := range flagEnvSecret {
				varname := strings.Replace(strings.ToUpper(secret), "-", "_", -1)
				secrets = append(secrets, varname+"="+secret+":latest")
			}
			gcloud = append(gcloud, "--set-secrets", strings.Join(secrets, ","))
		}
		if len(flagCloudSQL) > 0 {
			var instances []string
			for _, instance := range flagCloudSQL {
				instances = append(instances, fmt.Sprintf("%s:%s:%s", flagProject, flagRegion, instance))
			}
			gcloud = append(gcloud, "--set-cloudsql-instances", strings.Join(instances, ","))
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

		return nil
	}
}
