package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	os.Setenv("GITHUB_REF", "foo-version")

	content, err := ioutil.ReadFile("test.jsonnet")
	require.NoError(t, err)

	nativeFuncs := []*jsonnet.NativeFunction{
		{
			Name:   "sentry",
			Params: []ast.Identifier{"name"},
			Func: func(args []interface{}) (interface{}, error) {
				return args[0].(string) + "-dsn", nil
			},
		},
	}
	buf, err := runScript("test", content, nativeFuncs)
	require.NoError(t, err)

	fmt.Println(buf.String())

	golden, err := ioutil.ReadFile("golden.json")
	require.NoError(t, err)
	require.JSONEq(t, string(golden), buf.String())
}
