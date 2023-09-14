package debug

import (
	"fmt"
	"net"

	"github.com/altipla-consulting/errors"
	"github.com/spf13/cobra"
)

var cmdHost = &cobra.Command{
	Use:     "host",
	Short:   "Debug the address of a remote host.",
	Example: "wave debug host example",
	Args:    cobra.ExactArgs(1),
}

func init() {
	cmdHost.RunE = func(cmd *cobra.Command, args []string) error {
		ips, err := net.DefaultResolver.LookupHost(cmd.Context(), args[0])
		if err != nil {
			return errors.Trace(err)
		}
		for _, ip := range ips {
			fmt.Println(ip)
		}
		return nil
	}
}
