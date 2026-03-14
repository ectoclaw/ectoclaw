package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal"
)


func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			printVersion()
		},
	}

	return cmd
}

func printVersion() {
	fmt.Println(internal.GetVersion())
}
