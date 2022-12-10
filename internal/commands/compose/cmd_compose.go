package compose

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/atlassian/go-sentry-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/internal/query"
)

var (
	flagSentry string
)

func init() {
	Cmd.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	Cmd.MarkPersistentFlagRequired("sentry")
}

var Cmd = &cobra.Command{
	Use:     "compose",
	Short:   "Deploy with Docker Compose through SSH to a remote machine.",
	Example: "wave compose foo-1",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		logger := log.WithField("machine", args[0])

		logger.WithField("version", query.Version()).Info("Deploy to remote machine with Docker Compose")

		client, err := sentry.NewClient(os.Getenv("SENTRY_AUTH_TOKEN"), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}

		org := sentry.Organization{
			Slug: apiString("altipla-consulting"),
		}
		keys, err := client.GetClientKeys(org, sentry.Project{Slug: apiString(flagSentry)})
		if err != nil {
			return errors.Trace(err)
		}

		keygen := exec.Command("ssh-keygen", "-F", args[0])
		keygen.Stderr = os.Stderr
		if err := keygen.Run(); err != nil {
			if exit := new(exec.ExitError); !errors.As(err, &exit) || exit.ExitCode() != 1 {
				return errors.Trace(err)
			}

			// Error is expected, the host is not in the known hosts.
		} else {
			// Remove the stored host, could be outdated.
			logger.Info("Removing old stored host authentication")
			rm := exec.Command("ssh-keygen", "-R", args[0])
			rm.Stderr = os.Stderr
			if err := rm.Run(); err != nil {
				return errors.Trace(err)
			}
		}

		logger.Info("Downloading SSH key from the remote machine")
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Trace(err)
		}
		f, err := os.OpenFile(filepath.Join(home, ".ssh", "known_hosts"), os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Trace(err)
		}
		defer f.Close()
		keyscan := exec.Command("ssh-keyscan", args[0])
		keyscan.Stdout = f
		keyscan.Stderr = os.Stderr
		if err := keyscan.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Sending container changes to the remote machine")
		compose := exec.Command("docker", "compose", "-f", "docker-compose.prod.yml", "up", "-d")
		compose.Stderr = os.Stderr
		compose.Stdout = os.Stdout
		compose.Env = os.Environ()
		compose.Env = append(compose.Env, "DOCKER_HOST=ssh://jenkins@"+args[0])
		compose.Env = append(compose.Env, "VERSION="+query.Version())
		compose.Env = append(compose.Env, "SENTRY_DSN="+keys[0].DSN.Public)
		if err := compose.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}

func apiString(s string) *string {
	return &s
}
