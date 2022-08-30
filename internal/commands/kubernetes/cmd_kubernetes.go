package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atlassian/go-sentry-api"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libs.altipla.consulting/errors"

	"github.com/altipla-consulting/wave/embed"
	"github.com/altipla-consulting/wave/internal/query"
)

var (
	flagFilter   string
	flagEnv      []string
	flagIncludes []string
	flagApply    bool
)

func init() {
	Cmd.PersistentFlags().StringVarP(&flagFilter, "filter", "f", "", "Filter top level items when generating items.")
	Cmd.PersistentFlags().StringSliceVarP(&flagEnv, "env", "e", nil, "Set external variables.")
	Cmd.PersistentFlags().StringSliceVarP(&flagIncludes, "include", "i", nil, "Directories to include when running the jsonnet script.")
	Cmd.PersistentFlags().BoolVar(&flagApply, "apply", false, "Apply the output to the Kubernetes cluster instead of printing it.")
}

var Cmd = &cobra.Command{
	Use:     "kubernetes",
	Short:   "Run a jsonnet script and deploy the result to Kubernetes.",
	Example: "wave kubernetes k8s/deploy.jsonnet",
	Args:    cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		content, err := ioutil.ReadFile(args[0])
		if err != nil {
			return errors.Trace(err)
		}

		sentryClient, err := sentry.NewClient(os.Getenv("SENTRY_AUTH_TOKEN"), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}
		nativeFuncs := []*jsonnet.NativeFunction{
			nativeFuncSentry(sentryClient),
		}

		result, err := runScript(args[0], content, nativeFuncs)
		if err != nil {
			return errors.Trace(err)
		}

		if !flagApply {
			fmt.Println(result.String())
			return nil
		}

		log.WithFields(log.Fields{
			"filename": args[0],
			"version":  query.Version(),
		}).Info("Deploy generated file")

		apply := exec.Command("kubectl", "apply", "-f", "-")
		apply.Stdout = os.Stdout
		apply.Stderr = os.Stderr
		apply.Stdin = result
		if err := apply.Run(); err != nil {
			return errors.Trace(err)
		}

		return nil
	},
}

func runScript(filename string, content []byte, nativeFuncs []*jsonnet.NativeFunction) (*bytes.Buffer, error) {
	dir, err := ioutil.TempDir("", "wave")
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer os.RemoveAll(dir)
	if err := ioutil.WriteFile(filepath.Join(dir, "wave.jsonnet"), embed.Wave, 0600); err != nil {
		return nil, errors.Trace(err)
	}

	vm := jsonnet.MakeVM()
	vm.Importer(&jsonnet.FileImporter{
		JPaths: append(flagIncludes, ".", dir),
	})
	for _, f := range nativeFuncs {
		vm.NativeFunction(f)
	}

	vm.ExtVar("version", query.Version())

	for _, v := range flagEnv {
		parts := strings.Split(v, "=")
		if len(parts) != 2 {
			return nil, errors.Errorf("malformed environment variable: %s", v)
		}
		vm.ExtVar(parts[0], parts[1])
	}

	output, err := vm.EvaluateSnippet(filename, string(content))
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, errors.Trace(err)
	}
	list := &k8sList{
		APIVersion: "v1",
		Kind:       "List",
	}
	var filters []string
	if flagFilter != "" {
		filters = strings.Split(flagFilter, ".")
	}
	extractItems(list, filters, result)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(list); err != nil {
		return nil, errors.Trace(err)
	}

	return &buf, nil
}

func apiString(s string) *string {
	return &s
}

func nativeFuncSentry(client *sentry.Client) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "sentry",
		Params: []ast.Identifier{"name"},
		Func: func(args []interface{}) (interface{}, error) {
			org := sentry.Organization{
				Slug: apiString("altipla-consulting"),
			}
			keys, err := client.GetClientKeys(org, sentry.Project{Slug: apiString(args[0].(string))})
			if err != nil {
				return nil, errors.Trace(err)
			}

			return keys[0].DSN.Public, nil
		},
	}
}

type k8sList struct {
	APIVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	Items      []interface{} `json:"items"`
}

func extractItems(list *k8sList, filter []string, v interface{}) {
	switch v := v.(type) {
	case map[string]interface{}:
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := v[key]
			if ent, ok := child.(map[string]interface{}); ok {
				if _, ok := ent["apiVersion"]; ok {
					if len(filter) == 0 || filter[0] == key {
						list.Items = append(list.Items, ent)
					}
					continue
				}
			}

			switch {
			case len(filter) == 0:
				extractItems(list, filter, child)
			case filter[0] == key:
				extractItems(list, filter[1:], child)
			}
		}

	case nil:
		return

	default:
		panic(fmt.Sprintf("should not reach here: %#v", v))
	}
}
