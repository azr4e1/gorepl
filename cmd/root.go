package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const Version = "v0.4.1"

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
	logPathVar        string
	portVar           int
	portVarSend       int
	forceVar          bool
	addressVarSend    string
	allConnections    bool
	jsonVar           bool
	connectionVar     connectionFlag = "uds"
	connectionVarSend connectionFlag = "uds"
	rootCmd                          = &cobra.Command{
		Use:     "gorepl",
		Short:   "A repl multiplexer",
		Long:    "gorepl allows you to spin up a repl and send lines to it from any terminal in the same directory",
		Version: Version,
	}
)

// completion for connection
func connectionCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"tcp", "uds", "namedpipe"}, cobra.ShellCompDirectiveNoFileComp
}

// completion for port
func portCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ports := []string{}
	allTCPconns, err := getAllConnections("tcp")
	if err != nil {
		return ports, cobra.ShellCompDirectiveNoFileComp
	}
	for _, c := range allTCPconns {
		_, port, err := net.SplitHostPort(c.Address)
		if err != nil {
			continue
		}
		ports = append(ports, port)
	}
	return ports, cobra.ShellCompDirectiveNoFileComp
}

func addressCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	addresses := []string{}
	if connectionVarSend.String() == "tcp" {
		return addresses, cobra.ShellCompDirectiveNoFileComp
	}
	allConnections, err := getAllConnections(connectionVarSend.String())
	if err != nil {
		return addresses, cobra.ShellCompDirectiveNoFileComp
	}
	for _, c := range allConnections {
		addresses = append(addresses, c.Address)
	}
	return addresses, cobra.ShellCompDirectiveNoFileComp
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
