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

var cmdDeployJob = &cobra.Command{
	Use:     "deploy-job",
	Aliases: []string{"deployjob", "buildjob", "build-job"},
	Short:   "Deploy the job to Azure Container Apps.",
	Example: "wave container-apps deploy-job foo --subscription 1234-5678-9012-3456 --resource-group foo-rg",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagRepo, flagSubscription, flagResourceGroup string
	var flagSentry string
	cmdDeployJob.Flags().StringVar(&flagRepo, "repo", "", "Azure Container Registry repository name where the container will be stored.")
	cmdDeployJob.Flags().StringVar(&flagSubscription, "subscription", "", "Azure subscription ID.")
	cmdDeployJob.Flags().StringVar(&flagResourceGroup, "resource-group", "", "Azure resource group where the container has been stored. Use `wave acr` to upload it previously.")
	cmdDeployJob.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdDeployJob.MarkFlagRequired("repo")
	cmdDeployJob.MarkFlagRequired("subscription")
	cmdDeployJob.MarkFlagRequired("resource-group")
	cmdDeployJob.MarkFlagRequired("sentry")

	cmdDeployJob.RunE = func(cmd *cobra.Command, args []string) error {
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
