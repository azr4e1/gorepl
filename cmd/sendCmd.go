package cmd

import (
	"fmt"
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
		Args:  cobra.MaximumNArgs(1),
	}
)

func Send(command *cobra.Command, args []string) {
	if len(args) == 0 {
		command.Help()
		os.Exit(0)
	}
	tempDirPath, err := internals.GetNPipePathCurDir()
	if err != nil {
		cobra.CheckErr(err)
	}
	nPipe, err := internals.NewTempFifo(tempDirPath)
	if err != nil {
		cobra.CheckErr(err)
	}
	cmd := []byte(fmt.Sprintln(strings.Join(args, " ")))
	_, err = nPipe.Write(cmd)
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
