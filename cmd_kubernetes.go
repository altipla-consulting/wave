package main

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

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/altipla-consulting/wave/embed"
	"github.com/altipla-consulting/wave/internal/env"
	"github.com/altipla-consulting/wave/internal/query"
)

var cmdKubernetes = &cobra.Command{
	Use:     "kubernetes",
	Short:   "Run a jsonnet script and deploy the result to Kubernetes.",
	Example: "wave kubernetes k8s/deploy.jsonnet",
	Args:    cobra.ExactArgs(1),
}

func init() {
	var flagFilter string
	var flagEnv, flagIncludes []string
	var flagApply bool
	cmdKubernetes.PersistentFlags().StringVarP(&flagFilter, "filter", "f", "", "Filter top level items when generating items.")
	cmdKubernetes.PersistentFlags().StringSliceVarP(&flagEnv, "env", "e", nil, "Set external variables.")
	cmdKubernetes.PersistentFlags().StringSliceVarP(&flagIncludes, "include", "i", nil, "Directories to include when running the jsonnet script.")
	cmdKubernetes.PersistentFlags().BoolVar(&flagApply, "apply", false, "Apply the output to the Kubernetes cluster instead of printing it.")

	cmdKubernetes.RunE = func(command *cobra.Command, args []string) error {
		content, err := ioutil.ReadFile(args[0])
		if err != nil {
			return errors.Trace(err)
		}

		sentryClient, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}
		nativeFuncs := []*jsonnet.NativeFunction{
			nativeFuncSentry(sentryClient),
		}

		opts := RunOptions{
			NativeFuncs: nativeFuncs,
			Includes:    flagIncludes,
			Env:         flagEnv,
			Filter:      flagFilter,
		}
		result, err := runScript(args[0], content, opts)
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
	}
}

type RunOptions struct {
	NativeFuncs []*jsonnet.NativeFunction
	Includes    []string
	Env         []string
	Filter      string
}

func runScript(filename string, content []byte, opts RunOptions) (*bytes.Buffer, error) {
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
		JPaths: append(opts.Includes, ".", dir),
	})
	for _, f := range opts.NativeFuncs {
		vm.NativeFunction(f)
	}

	vm.ExtVar("version", query.Version())

	for _, v := range opts.Env {
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
	if opts.Filter != "" {
		filters = strings.Split(opts.Filter, ".")
	}
	extractItems(list, filters, result)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(list); err != nil {
		return nil, errors.Trace(err)
	}

	return &buf, nil
}

func nativeFuncSentry(client *sentry.Client) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "sentry",
		Params: []ast.Identifier{"name"},
		Func: func(args []interface{}) (interface{}, error) {
			org := sentry.Organization{
				Slug: sentryAPIString("altipla-consulting"),
			}
			keys, err := client.GetClientKeys(org, sentry.Project{Slug: sentryAPIString(args[0].(string))})
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
