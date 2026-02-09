package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

var (
	sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send data to repl",
		Run:   Send,
	}
)

func Send(command *cobra.Command, args []string) {
	if namedPipeVar {
		err := sendNamedPipe(command, args)
		if err != nil {
			cobra.CheckErr(err)
		}
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().BoolVarP(&namedPipeVar, "named-pipe", "n", true, "Connect to a named pipe")
}

func sendNamedPipe(command *cobra.Command, args []string) error {
	// get connection
	tempDirPath, err := internals.GetNPipePathCurDir()
	if err != nil {
		return err
	}
	nPipe, err := internals.NewTempFifo(tempDirPath)
	if err != nil {
		return errors.New("Couldn't connect to a named pipe. Are you sure your repl is running in this directory?")
	}
	defer nPipe.Close()

	// determine if we are getting piped
	fi, _ := os.Stdin.Stat()

	var lines []byte
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		lines, _ = io.ReadAll(os.Stdin)
	} else {
		if len(args) == 0 {
			command.Help()
			os.Exit(0)
		}
		lines = fmt.Appendln(nil, strings.Join(args, ""))
	}
	_, err = nPipe.Write(lines)
	return err
}
