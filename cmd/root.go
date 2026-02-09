package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const Version = "v0.1.0"

var (
	namedPipeVar bool
	force        bool
	rootCmd      = &cobra.Command{
		Use:     "gorepl",
		Short:   "A repl multiplexer",
		Long:    "gorepl allows you to spin up a repl and send lines to it from any terminal in the same directory",
		Version: Version,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
