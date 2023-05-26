package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdCompose = &cobra.Command{
	Use:     "compose",
	Short:   "Deploy with Docker Compose through SSH to a remote machine.",
	Example: "wave compose foo-1",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagSentry, flagFile string
	cmdCompose.Flags().StringVar(&flagSentry, "sentry", "", "Name of the sentry project to configure.")
	cmdCompose.Flags().StringVar(&flagFile, "file", "docker-compose.prod.yml", "Path to the Docker Compose file to deploy.")
	cmdCompose.MarkPersistentFlagRequired("sentry")

	cmdCompose.RunE = func(cmd *cobra.Command, args []string) error {
		logger := log.WithField("machine", args[0])
		logger.WithField("version", query.Version()).Info("Deploy to remote machine with Docker Compose")

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

		logger.Info("Downloading SSH key from the remote machine")
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh", "known_hosts")); err != nil && !os.IsNotExist(err) {
			return errors.Trace(err)
		} else if err == nil {
			if err := cleanHost(cmd.Context(), logger, args[0]); err != nil {
				return errors.Trace(err)
			}
			ips, err := net.DefaultResolver.LookupHost(cmd.Context(), args[0])
			if err != nil {
				return errors.Trace(err)
			}
			for _, ip := range ips {
				if err := cleanHost(cmd.Context(), logger, ip); err != nil {
					return errors.Trace(err)
				}
			}
		} else {
			if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0700); err != nil {
				return errors.Trace(err)
			}
		}
		f, err := os.OpenFile(filepath.Join(home, ".ssh", "known_hosts"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return errors.Trace(err)
		}
		defer f.Close()
		keyscan := exec.CommandContext(cmd.Context(), "ssh-keyscan", args[0])
		keyscan.Stdout = f
		keyscan.Stderr = os.Stderr
		if err := keyscan.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Building remote containers")
		build := exec.CommandContext(cmd.Context(), "docker", "compose", "-f", flagFile, "build")
		prepareComposeCommand(build, args[0], keys[0].DSN.Public)
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Sending container changes to the remote machine")
		up := exec.CommandContext(cmd.Context(), "docker", "compose", "-f", flagFile, "up", "-d")
		prepareComposeCommand(up, args[0], keys[0].DSN.Public)
		if err := up.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}

func prepareComposeCommand(cmd *exec.Cmd, machine, sentryDSN string) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "DOCKER_HOST=ssh://jenkins@"+machine)
	cmd.Env = append(cmd.Env, "VERSION="+query.Version())
	cmd.Env = append(cmd.Env, "SENTRY_DSN="+sentryDSN)
}

func cleanHost(ctx context.Context, logger *log.Entry, host string) error {
	keygen := exec.CommandContext(ctx, "ssh-keygen", "-F", host)
	keygen.Stderr = os.Stderr
	if err := keygen.Run(); err != nil {
		if exit := new(exec.ExitError); !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return errors.Trace(err)
		}

		// Error is expected, the host is not in the known hosts.
		return nil
	} else {
		// Remove the stored host, could be outdated.
		logger.WithField("host", host).Info("Removing old stored host authentication")
		rm := exec.Command("ssh-keygen", "-R", host)
		rm.Stderr = os.Stderr
		if err := rm.Run(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
