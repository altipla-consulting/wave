package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/gerrit"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdPreview = &cobra.Command{
	Use:     "preview",
	Short:   "Send preview URLs as a comment to Gerrit.",
	Example: "wave preview --cloud-run my-app",
	Args:    cobra.NoArgs,
}

func init() {
	var flagProject, flagTag, flagRegion string
	var flagCloudRun, flagNetlify, flagCloudflare []string
	cmdPreview.PersistentFlags().StringVar(&flagProject, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	cmdPreview.PersistentFlags().StringVar(&flagTag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	cmdPreview.PersistentFlags().StringSliceVar(&flagCloudRun, "cloud-run", nil, "Cloud Run applications. Format: `local-name:cloud-run-name`.")
	cmdPreview.PersistentFlags().StringSliceVar(&flagNetlify, "netlify", nil, "Netlify applications. Format: `local-name:netlify-name`.")
	cmdPreview.PersistentFlags().StringVar(&flagRegion, "region", "europe-west1", "Region where resources will be hosted.")
	cmdPreview.PersistentFlags().StringSliceVar(&flagCloudflare, "cloudflare", nil, "Cloudflare applications. Format: `local-name:cloudflare-name`.")

	cmdPreview.RunE = func(command *cobra.Command, args []string) error {
		if flagProject == "" && len(flagCloudRun) > 0 {
			flagProject = env.GoogleProject()
		}

		if len(flagCloudRun) == 0 && len(flagNetlify) == 0 && len(flagCloudflare) == 0 {
			return errors.Errorf("pass --cloud-run, --netlify or --cloudflare applications as arguments")
		}

		var runSuffix string
		if len(flagCloudRun) > 0 {
			_, remote, err := splitName(flagCloudRun[0])
			if err != nil {
				return errors.Trace(err)
			}
			suffixcmd := exec.Command(
				"gcloud",
				"run", "services", "describe",
				remote,
				"--format", "value(status.url)",
				"--region", flagRegion,
				"--project", flagProject,
			)
			output, err := suffixcmd.CombinedOutput()
			if err != nil {
				slog.Error(string(output))
				return errors.Trace(err)
			}
			u, err := url.Parse(strings.TrimSpace(string(output)))
			if err != nil {
				return errors.Trace(err)
			}
			parts := strings.Split(strings.Split(u.Host, ".")[0], "-")
			runSuffix = parts[len(parts)-2]
		}

		var previews []string
		host := query.VersionHostname(flagTag)
		for _, cr := range flagCloudRun {
			local, remote, err := splitName(cr)
			if err != nil {
				return errors.Trace(err)
			}
			previews = append(previews, local+" :: https://"+host+"---"+remote+"-"+runSuffix+"-ew.a.run.app/")
		}
		for _, netlify := range flagNetlify {
			local, remote, err := splitName(netlify)
			if err != nil {
				return errors.Trace(err)
			}
			previews = append(previews, local+" :: https://"+host+"--"+remote+".netlify.app/")
		}
		for _, cf := range flagCloudflare {
			local, remote, err := splitName(cf)
			if err != nil {
				return errors.Trace(err)
			}
			previews = append(previews, local+" :: https://"+host+"."+remote+".pages.dev/")
		}

		slog.Info("Send preview URLs as a Gerrit comment")
		for _, preview := range previews {
			fmt.Println(preview)
		}
		if err := gerrit.Comment("Previews deployed at:\n" + strings.Join(previews, "\n")); err != nil {
			return errors.Trace(err)
		}

		return nil
	}
}

func splitName(name string) (string, string, error) {
	switch parts := strings.Split(name, ":"); len(parts) {
	case 1:
		return parts[0], parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", errors.Errorf("application name has wrong format: %s", name)
	}
}
