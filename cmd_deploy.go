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

	var flagProject, flagRegion string
	var flagMemory string
	var flagServiceAccount string
	var flagSentry string
	var flagVolumeSecret, flagEnvSecret []string
	var flagEnv []string
	var flagTag string
	var flagAlwaysOn bool
	var flagCloudSQL []string
	var flagConcurrency int64
	cmdDeploy.Flags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdDeploy.Flags().StringVar(&flagMemory, "memory", "", "Memory available inside the Cloud Run application. Default: 256Mi.")
	cmdDeploy.Flags().StringVar(&flagServiceAccount, "service-account", "", "Service account. Defaults to one with the name of the application.")
	cmdDeploy.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdDeploy.Flags().StringSliceVar(&flagVolumeSecret, "volume-secret", nil, "Secrets to mount as volumes.")
	cmdDeploy.Flags().StringSliceVar(&flagEnvSecret, "env-secret", nil, "Secrets to mount as environment variables.")
	cmdDeploy.Flags().StringSliceVar(&flagEnv, "env", nil, "Custom environment variables to define as `KEY=value` pairs.")
	cmdDeploy.Flags().StringVar(&flagTag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	cmdDeploy.Flags().BoolVar(&flagAlwaysOn, "always-on", false, "App will always have CPU even if it's in the background without requests.")
	cmdDeploy.Flags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdDeploy.Flags().StringSliceVar(&flagCloudSQL, "cloudsql", nil, "CloudSQL instances to connect to. Only the name.")
	cmdDeploy.Flags().Int64Var(&flagConcurrency, "concurrency", 50, "Maximum number of concurrent requests.")
	cmdDeploy.MarkFlagRequired("sentry")

	cmdDeploy.RunE = func(command *cobra.Command, args []string) error {
		app := args[0]

		if flagProject == "" {
			flagProject = env.GoogleProject()
		}
		if flagServiceAccount == "" {
			flagServiceAccount = app
		}
		if flagMemory == "" {
			if flagAlwaysOn {
				flagMemory = "512Mi"
			} else {
				flagMemory = "256Mi"
			}
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
			"beta", "run", "deploy",
			app,
			"--image", "eu.gcr.io/" + flagProject + "/" + app + ":" + version,
			"--region", flagRegion,
			"--platform", "managed",
			"--concurrency", fmt.Sprintf("%d", flagConcurrency),
			"--timeout", "60s",
			"--service-account", flagServiceAccount + "@" + flagProject + ".iam.gserviceaccount.com",
			"--memory", flagMemory,
			"--set-env-vars", strings.Join(env, ","),
			"--labels", "app=" + app,
		}
		if len(flagVolumeSecret) > 0 || len(flagEnvSecret) > 0 {
			var secrets []string
			for _, secret := range flagVolumeSecret {
				if secret == "ravendb-client-credentials" {
					secrets = append(secrets, "/etc/secrets/"+secret+"="+secret+":latest")
				}
				secrets = append(secrets, "/etc/secrets-v2/"+secret+"/value="+secret+":latest")
			}
			for _, secret := range flagEnvSecret {
				varname := strings.Replace(strings.ToUpper(secret), "-", "_", -1)
				secrets = append(secrets, varname+"="+secret+":latest")
			}
			gcloud = append(gcloud, "--set-secrets", strings.Join(secrets, ","))
		}
		if tag := query.VersionHostname(flagTag); tag != "" {
			if !query.IsRelease() {
				gcloud = append(gcloud, "--no-traffic")
				gcloud = append(gcloud, "--max-instances", "1")
			}
			gcloud = append(gcloud, "--tag", tag)
		} else {
			gcloud = append(gcloud, "--max-instances", "20")
		}
		if flagAlwaysOn {
			gcloud = append(gcloud, "--no-cpu-throttling")
		} else {
			gcloud = append(gcloud, "--cpu-throttling")
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
				"--region", "europe-west1",
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
