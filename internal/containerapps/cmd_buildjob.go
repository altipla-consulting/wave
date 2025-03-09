package containerapps

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdBuildJob = &cobra.Command{
	Use:     "build-job",
	Aliases: []string{"buildjob"},
	Short:   "Deploy the job to Azure Container Apps.",
	Example: "wave container-apps build-job foo --subscription 1234-5678-9012-3456 --resource-group foo-rg",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagRepo, flagSubscription, flagResourceGroup string
	var flagSentry string
	cmdBuildJob.Flags().StringVar(&flagRepo, "repo", "", "Azure Container Registry repository name where the container will be stored.")
	cmdBuildJob.Flags().StringVar(&flagSubscription, "subscription", "", "Azure subscription ID.")
	cmdBuildJob.Flags().StringVar(&flagResourceGroup, "resource-group", "", "Azure resource group where the container has been stored. Use `wave acr` to upload it previously.")
	cmdBuildJob.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdBuildJob.MarkFlagRequired("repo")
	cmdBuildJob.MarkFlagRequired("subscription")
	cmdBuildJob.MarkFlagRequired("resource-group")
	cmdBuildJob.MarkFlagRequired("sentry")

	cmdBuildJob.RunE = func(cmd *cobra.Command, args []string) error {
		app := args[0]

		version := query.VersionImageTag(cmd.Context())
		logger := slog.With(slog.String("name", app), slog.String("version", version))
		logger.Info("Deploy app")

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

		auth := exec.CommandContext(cmd.Context(), "az", "account", "set", "--subscription", flagSubscription)
		auth.Stdout = os.Stdout
		auth.Stderr = os.Stderr
		if err := auth.Run(); err != nil {
			return errors.Trace(err)
		}

		az := []string{
			"containerapp", "job", "update",
			"--name", app,
			"--resource-group", flagResourceGroup,
			"--image", fmt.Sprintf("%s.azurecr.io/%s:%s", flagRepo, app, version),
			"--set-env-vars", fmt.Sprintf("VERSION=%s", version), fmt.Sprintf("SENTRY_DSN=%s", keys[0].DSN.Public),
		}
		deploy := exec.CommandContext(cmd.Context(), "az", az...)
		deploy.Stdout = os.Stdout
		deploy.Stderr = os.Stderr
		if err := deploy.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
