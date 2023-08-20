package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdLightsail = &cobra.Command{
	Use:     "lightsail",
	Short:   "Deploy with a new Lightsail Containers application.",
	Example: "wave lightsail",
	Args:    cobra.NoArgs,
}

type containerConfig struct {
	ServiceName string `json:"serviceName"`
}

func init() {
	var flagRepo, flagFile, flagRegion string
	cmdLightsail.Flags().StringVar(&flagRepo, "repo", "", "ECR repository name where the container is stored.")
	cmdLightsail.Flags().StringVar(&flagFile, "file", "containers.prod.json", "Path to the JSON file to deploy.")
	cmdLightsail.Flags().StringVar(&flagRegion, "region", "eu-west-1", "AWS region where the container is stored.")
	cmdLightsail.MarkFlagRequired("repo")

	cmdLightsail.RunE = func(cmd *cobra.Command, args []string) error {
		content, err := os.ReadFile(flagFile)
		if err != nil {
			return errors.Trace(err)
		}
		var config containerConfig
		if err := json.Unmarshal(content, &config); err != nil {
			return errors.Trace(err)
		}

		logger := log.WithField("machine", config.ServiceName)
		logger.WithField("version", query.Version()).Info("Deploy to Lightsail Containers")

		var mapErr error
		var mapFn = func(placeholder string) string {
			switch {
			case placeholder == "VERSION":
				return query.Version()

			case placeholder == "REPO":
				return flagRepo

			case strings.HasPrefix(placeholder, "SENTRY_DSN("):
				client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
				if err != nil {
					mapErr = errors.Trace(err)
					return "[ERROR]"
				}
				project := strings.TrimSuffix(strings.TrimPrefix(placeholder, "SENTRY_DSN("), ")")
				org := sentry.Organization{
					Slug: sentryAPIString("altipla-consulting"),
				}
				keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(project)})
				if err != nil {
					mapErr = errors.Trace(err)
					return "[ERROR]"
				}
				return keys[0].DSN.Public

			default:
				mapErr = errors.Errorf("unknown environment expansion: %s", placeholder)
				return "[ERROR]"
			}
		}
		content = []byte(os.Expand(string(content), mapFn))
		if mapErr != nil {
			return errors.Trace(mapErr)
		}

		tmpFile, err := os.CreateTemp("", "*.containers.prod.json")
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := tmpFile.Write(content); err != nil {
			return errors.Trace(err)
		}
		log.WithField("file", tmpFile.Name()).Info("Using deployment file")

		create := []string{
			"aws", "lightsail",
			"create-container-service-deployment",
			"--service-name", config.ServiceName,
			"--cli-input-json", "file://" + tmpFile.Name(),
			"--region", flagRegion,
			"--no-cli-pager",
		}
		createCmd := exec.CommandContext(cmd.Context(), create[0], create[1:]...)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		if err := createCmd.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
