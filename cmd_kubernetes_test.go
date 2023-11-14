package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	os.Setenv("WAVE_VERSION", "foo-version")

	content, err := ioutil.ReadFile("testdata/test.jsonnet")
	require.NoError(t, err)

	opts := RunOptions{
		NativeFuncs: []*jsonnet.NativeFunction{
			{
				Name:   "sentry",
				Params: []ast.Identifier{"name"},
				Func: func(args []interface{}) (interface{}, error) {
					return args[0].(string) + "-dsn", nil
				},
			},
		},
	}
	buf, err := runScript(context.Background(), "test", content, opts)
	require.NoError(t, err)

	fmt.Println(buf.String())

	golden, err := ioutil.ReadFile("testdata/golden.json")
	require.NoError(t, err)
	require.JSONEq(t, string(golden), buf.String())
}
