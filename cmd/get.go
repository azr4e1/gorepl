package cmd

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

var (
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get the active connection in this directory",
		Run:   Get,
		Args:  cobra.NoArgs,
	}
)

func Get(command *cobra.Command, args []string) {
	err := getNamedPipe(os.Stdout)
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func getNamedPipe(output io.Writer) error {
	tempDir, err := internals.GetNPipePathCurDir()
	if err != nil {
		return err
	}
	nPipePath := path.Join(tempDir, internals.NPipeName)
	ok, err := internals.Exists(nPipePath)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(output, "There is no active connection at this location")
		return nil
	}
	fmt.Fprintf(output, "There is an active connection at this location, with address: '%s'\n", nPipePath)
	return nil
}
