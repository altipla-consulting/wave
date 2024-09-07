package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/altipla-consulting/errors"
	"github.com/atlassian/go-sentry-api"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/joho/godotenv"
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
	var flagApply, flagDisableSentry bool
	cmdKubernetes.Flags().StringVarP(&flagFilter, "filter", "f", "", "Filter top level items when generating items.")
	cmdKubernetes.Flags().StringSliceVarP(&flagEnv, "env", "e", nil, "Set external variables.")
	cmdKubernetes.Flags().StringSliceVarP(&flagIncludes, "include", "i", nil, "Directories to include when running the jsonnet script.")
	cmdKubernetes.Flags().BoolVar(&flagApply, "apply", false, "Apply the output to the Kubernetes cluster instead of printing it.")
	cmdKubernetes.Flags().BoolVar(&flagDisableSentry, "disable-sentry", false, "Disable Sentry configurations allowing a quick break-glass deployment.")

	cmdKubernetes.RunE = func(command *cobra.Command, args []string) error {
		opts := RunOptions{
			NativeFuncs: []*jsonnet.NativeFunction{
				nativeFuncSentry(flagDisableSentry),
				nativeFuncEnvFile(),
			},
			Includes: flagIncludes,
			Env:      flagEnv,
			Filter:   flagFilter,
		}
		result, err := runScript(command.Context(), args[0], opts)
		if err != nil {
			return errors.Trace(err)
		}

		if !flagApply {
			fmt.Println(result.String())
			return nil
		}

		log.WithFields(log.Fields{
			"filename": args[0],
			"version":  query.Version(command.Context()),
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

type customImporter struct {
	file *jsonnet.FileImporter
	mem  *jsonnet.MemoryImporter
}

func (c *customImporter) Import(importedFrom string, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	if importedPath == "wave.jsonnet" {
		return c.mem.Import(importedFrom, importedPath)
	}
	return c.file.Import(importedFrom, importedPath)
}

func runScript(ctx context.Context, filename string, opts RunOptions) (*bytes.Buffer, error) {
	vm := jsonnet.MakeVM()
	vm.Importer(&customImporter{
		file: &jsonnet.FileImporter{
			JPaths: append(opts.Includes, "."),
		},
		mem: &jsonnet.MemoryImporter{
			Data: map[string]jsonnet.Contents{
				"wave.jsonnet": jsonnet.MakeContentsRaw(embed.Wave),
			},
		},
	})
	for _, f := range opts.NativeFuncs {
		vm.NativeFunction(f)
	}
	vm.ExtVar("version", query.Version(ctx))
	vm.ExtVar("image-tag", query.VersionImageTag(ctx))

	for _, v := range opts.Env {
		parts := strings.Split(v, "=")
		if len(parts) != 2 {
			return nil, errors.Errorf("malformed environment variable: %s", v)
		}
		vm.ExtVar(parts[0], parts[1])
	}

	output, err := vm.EvaluateFile(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result any
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

func nativeFuncSentry(disableSentry bool) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "sentry",
		Params: []ast.Identifier{"name"},
		Func: func(args []any) (any, error) {
			if os.Getenv("SENTRY_AUTH_TOKEN") == "" {
				if disableSentry {
					return "", nil
				}

				// Panic with the correct error.
				env.SentryAuthToken()
			}

			client, err := sentry.NewClient(env.SentryAuthToken(), nil, nil)
			if err != nil {
				return nil, errors.Trace(err)
			}

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

func nativeFuncEnvFile() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "envfile",
		Params: []ast.Identifier{"filename"},
		Func: func(args []any) (any, error) {
			m, err := godotenv.Read(args[0].(string))
			if err != nil {
				return nil, errors.Trace(err)
			}
			res := make(map[string]any)
			for k, v := range m {
				res[k] = base64.URLEncoding.EncodeToString([]byte(v))
			}
			return res, nil
		},
	}
}

type k8sList struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Items      []any  `json:"items"`
}

func extractItems(list *k8sList, filter []string, v any) {
	switch v := v.(type) {
	case map[string]any:
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := v[key]
			if ent, ok := child.(map[string]any); ok {
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
