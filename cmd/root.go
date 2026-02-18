package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const Version = "v0.1.1"

type connectionFlag string

func (c *connectionFlag) String() string { return string(*c) }
func (c *connectionFlag) Type() string   { return "connection" }
func (c *connectionFlag) Set(v string) error {
	allowed := []string{"uds", "tcp", "namedpipe"}
	for _, a := range allowed {
		if v == a {
			*c = connectionFlag(a)
			return nil
		}
	}
	return fmt.Errorf("must be one of %s", strings.Join(allowed, ", "))
}

var (
	namedPipeSendVar bool
	logPathVar       string
	port             int
	forceVar         bool
	jsonVar          bool
	rootCmd          = &cobra.Command{
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
