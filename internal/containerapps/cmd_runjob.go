package containerapps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/altipla-consulting/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdRunJob = &cobra.Command{
	Use:     "run-job",
	Aliases: []string{"runjob"},
	Short:   "Run a job in Azure Container Apps and wait for completion",
	Example: "wave container-apps run-job foo --subscription 1234-5678-9012-3456 --resource-group foo-rg",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagSubscription, flagResourceGroup string
	cmdRunJob.Flags().StringVar(&flagSubscription, "subscription", "", "Azure subscription ID.")
	cmdRunJob.Flags().StringVar(&flagResourceGroup, "resource-group", "", "Azure resource group where the job is located.")
	cmdRunJob.MarkFlagRequired("subscription")
	cmdRunJob.MarkFlagRequired("resource-group")

	cmdRunJob.RunE = func(cmd *cobra.Command, args []string) error {
		jobName := args[0]

		logger := log.WithFields(log.Fields{
			"name": jobName,
		})
		logger.Info("Running job")

		auth := exec.CommandContext(cmd.Context(), "az", "account", "set", "--subscription", flagSubscription)
		auth.Stdout = os.Stdout
		auth.Stderr = os.Stderr
		if err := auth.Run(); err != nil {
			return errors.Trace(err)
		}

		az := []string{
			"containerapp", "job", "start",
			"--name", jobName,
			"--resource-group", flagResourceGroup,
		}
		run := exec.CommandContext(cmd.Context(), "az", az...)
		run.Stdout = os.Stdout
		run.Stderr = os.Stderr
		if err := run.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Waiting for job completion")
		wait := []string{
			"containerapp", "job", "execution", "list",
			"--name", jobName,
			"--resource-group", flagResourceGroup,
			"--query", "sort_by([].{status: properties.status, startTime: properties.startTime}, &startTime)[-1].status",
			"--output", "tsv",
		}
		for {
			check := exec.CommandContext(cmd.Context(), "az", wait...)
			check.Stderr = os.Stderr
			output, err := check.Output()
			if err != nil {
				return errors.Trace(err)
			}

			status := strings.TrimSpace(string(output))
			logger.WithField("status", status).Debug("Job status")

			switch status {
			case "Running":
				fmt.Print(".")
				os.Stdout.Sync()
			case "Succeeded":
				fmt.Println()
				logger.Info("Job completed successfully")
				return nil
			case "Failed":
				fmt.Println()
				return errors.Errorf("Container Apps Job failed to complete. Read the Azure Portal logs for more information.")
			default:
				fmt.Println()
				return errors.Errorf("unknown job status %q when reading from wave", status)
			}

			time.Sleep(3 * time.Second)
		}
	}
}
