package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
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

	cmdCompose.RunE = func(cmd *cobra.Command, args []string) error {
		logger := slog.With(slog.String("machine", args[0]))
		logger.Info("Deploy to remote machine with Docker Compose", slog.String("version", query.Version(cmd.Context())))

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

		content, err := os.ReadFile(flagFile)
		if err != nil {
			return errors.Trace(err)
		}
		var mapErr error
		var mapFn = func(placeholder string) string {
			val, err := replaceEnv(cmd.Context(), flagSentry, placeholder)
			if err != nil {
				mapErr = errors.Trace(err)
				return "[ERROR]"
			}
			return val
		}
		content = []byte(os.Expand(string(content), mapFn))
		if mapErr != nil {
			return errors.Trace(mapErr)
		}

		tmpFile := filepath.Join(filepath.Dir(flagFile), "docker-compose.prod-tmpl.yml")
		if err := os.WriteFile(tmpFile, content, 0600); err != nil {
			return errors.Trace(err)
		}
		defer os.Remove(tmpFile)

		logger.Info("Building remote containers")
		build := exec.CommandContext(cmd.Context(), "docker", "compose", "-f", tmpFile, "build")
		build.Stderr = os.Stderr
		build.Stdout = os.Stdout
		build.Env = os.Environ()
		build.Env = append(build.Env, "DOCKER_HOST=ssh://jenkins@"+args[0])
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		logger.Info("Sending container changes to the remote machine")
		up := exec.CommandContext(cmd.Context(), "docker", "compose", "-f", tmpFile, "up", "-d", "--remove-orphans")
		up.Stderr = os.Stderr
		up.Stdout = os.Stdout
		up.Env = os.Environ()
		up.Env = append(up.Env, "DOCKER_HOST=ssh://jenkins@"+args[0])
		if err := up.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}

func cleanHost(ctx context.Context, logger *slog.Logger, host string) error {
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
		logger.Info("Removing old stored host authentication", slog.String("host", host))
		rm := exec.Command("ssh-keygen", "-R", host)
		rm.Stderr = os.Stderr
		if err := rm.Run(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func replaceEnv(ctx context.Context, flagSentry, placeholder string) (string, error) {
	switch {
	case placeholder == "VERSION":
		return query.Version(ctx), nil

	case placeholder == "IMAGE_TAG":
		return query.VersionImageTag(ctx), nil

	case placeholder == "SENTRY_DSN":
		if flagSentry == "" {
			return "", errors.Errorf("missing --sentry flag")
		}
		client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
		if err != nil {
			return "", errors.Trace(err)
		}
		org := sentry.Organization{
			Slug: sentryAPIString("altipla"),
		}
		keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(flagSentry)})
		if err != nil {
			return "", errors.Trace(err)
		}
		return keys[0].DSN.Public, nil

	case strings.HasPrefix(placeholder, "SENTRY_DSN("):
		client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
		if err != nil {
			return "", errors.Trace(err)
		}
		project := strings.TrimSuffix(strings.TrimPrefix(placeholder, "SENTRY_DSN("), ")")
		org := sentry.Organization{
			Slug: sentryAPIString("altipla"),
		}
		keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(project)})
		if err != nil {
			return "", errors.Trace(err)
		}
		return keys[0].DSN.Public, nil

	case os.Getenv(placeholder) != "":
		return os.Getenv(placeholder), nil

	default:
		return "", errors.Errorf("unknown environment expansion: %s", placeholder)
	}
}
