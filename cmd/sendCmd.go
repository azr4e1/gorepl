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
	if namedPipeSendVar {
		err := sendNamedPipe(command, args)
		if err != nil {
			cobra.CheckErr(err)
		}
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().BoolVarP(&namedPipeSendVar, "named-pipe", "n", true, "Connect to a named pipe")
}

func sendNamedPipe(command *cobra.Command, args []string) error {
	// get connection
	tempPath, err := internals.GetPathCurDir()
	if err != nil {
		return err
	}
	nPipe, err := internals.NewTempFifo(tempPath)
	if err != nil {
		return errors.New("Couldn't connect to a named pipe. Are you sure your repl is running in this directory?")
	}
	defer nPipe.Close()

	// determine if we are getting piped
	fi, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	var lines []byte
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		lines, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	} else {
		if len(args) == 0 {
			command.Help()
			os.Exit(0)
		}
		lines = fmt.Appendln(nil, strings.Join(args, " "))
	}
	_, err = nPipe.Write(preprocess(lines))
	return err
}

func preprocess(buf []byte) []byte {
	lines := strings.Split(string(buf), "\n")
	cleanLines := []string{}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		cleanLines = append(cleanLines, strings.TrimRight(l, "\n\t "))
	}
	bufString := strings.Join(cleanLines, "\n") + "\n"
	return []byte(bufString)
}
