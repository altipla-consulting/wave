package preview

import (
	"net/url"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/internal/gerrit"
)

type cmdFlags struct {
	Project  string
	Tag      string
	CloudRun []string
	Netlify  []string
	Region   string
}

var (
	flags cmdFlags
)

func init() {
	Cmd.PersistentFlags().StringVar(&flags.Project, "project", "", "Google Cloud project where the container will be stored. Defaults to the GOOGLE_PROJECT environment variable.")
	Cmd.PersistentFlags().StringVar(&flags.Tag, "tag", "", "Name of the revision included in the URL. Defaults to the Gerrit change and patchset.")
	Cmd.PersistentFlags().StringSliceVar(&flags.CloudRun, "cloud-run", nil, "Cloud Run applications. Format: `local-name:cloud-run-name`.")
	Cmd.PersistentFlags().StringSliceVar(&flags.Netlify, "netlify", nil, "Netlify applications. Format: `local-name:netlify-name`.")
	Cmd.PersistentFlags().StringVar(&flags.Region, "region", "europe-west1", "Region where resources will be hosted.")
}

var Cmd = &cobra.Command{
	Use:     "preview",
	Short:   "Send preview URLs as a comment to Gerrit.",
	Example: "wave preview --cloud-run my-app",
	RunE: func(command *cobra.Command, args []string) error {
		if flags.Project == "" {
			flags.Project = os.Getenv("GOOGLE_PROJECT")
		}

		if len(flags.CloudRun) == 0 && len(flags.Netlify) == 0 {
			return errors.Errorf("pass --cloud-run or --netlify applications as arguments")
		}

		if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" && flags.Tag == "" {
			flags.Tag = gerrit.Descriptor()
		}

		var suffix string
		if len(flags.CloudRun) > 0 {
			_, remote, err := splitName(flags.CloudRun[0])
			if err != nil {
				return errors.Trace(err)
			}
			suffixcmd := exec.Command(
				"gcloud",
				"run", "services", "describe",
				remote,
				"--format", "value(status.url)",
				"--region", flags.Region,
				"--project", flags.Project,
			)
			output, err := suffixcmd.CombinedOutput()
			if err != nil {
				log.Error(string(output))
				return errors.Trace(err)
			}
			u, err := url.Parse(strings.TrimSpace(string(output)))
			if err != nil {
				return errors.Trace(err)
			}
			parts := strings.Split(strings.Split(u.Host, ".")[0], "-")
			suffix = parts[len(parts)-2]
		}

		var previews []string
		for _, cr := range flags.CloudRun {
			local, remote, err := splitName(cr)
			if err != nil {
				return errors.Trace(err)
			}
			previews = append(previews, local+" :: https://"+flags.Tag+"---"+remote+"-"+suffix+"-ew.a.run.app/")
		}
		for _, netlify := range flags.Netlify {
			local, remote, err := splitName(netlify)
			if err != nil {
				return errors.Trace(err)
			}
			previews = append(previews, local+" :: https://"+flags.Tag+"--"+remote+".netlify.app/")
		}

		log.Info("Send preview URLs as a Gerrit comment")
		for _, preview := range previews {
			log.Println(preview)
		}

		if err := gerrit.Comment("Previews deployed at:\n" + strings.Join(previews, "\n")); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
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
