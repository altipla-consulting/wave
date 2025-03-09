package main

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/query"
)

var cmdNetlify = &cobra.Command{
	Use:     "netlify",
	Short:   "Deploy a site to Netlify.",
	Example: "wave netlify foo",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagTag, flagSource string
	cmdNetlify.PersistentFlags().StringVar(&flagTag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	cmdNetlify.PersistentFlags().StringVar(&flagSource, "source", "", "Source folder. Defaults to the name of the application.")

	cmdNetlify.RunE = func(command *cobra.Command, args []string) error {
		site := args[0]

		slog.Info("Get last commit message")
		cmd := exec.Command("git", "log", "-1", "--pretty=%B")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Trace(err)
		}
		var filtered []string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "Change-Id") {
				continue
			}
			filtered = append(filtered, line)
		}
		lastCommit := strings.TrimSpace(strings.Join(filtered, "\n"))

		slog.Info("Deploy Netlify site", slog.String("name", site), slog.String("version", query.Version(command.Context())))

		netlify := []string{
			"netlify",
			"deploy",
			"--dir", "dist",
			"--json",
			"--message", lastCommit,
		}
		if tag := query.VersionHostname(flagTag); tag != "" {
			netlify = append(netlify, "--alias", tag)
		}
		if query.IsRelease() {
			netlify = append(netlify, "--prod")
		}
		slog.Debug(strings.Join(netlify, " "))
		build := exec.Command(netlify[0], netlify[1:]...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Dir = flagSource
		if err := build.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}
