package cmd

import (
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
	tempDirPath, err := internals.GetNPipePathCurDir()
	if err != nil {
		cobra.CheckErr(err)
	}
	nPipe, err := internals.NewTempFifo(tempDirPath)
	if err != nil {
		cobra.CheckErr(err)
	}
	defer nPipe.Close()
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
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
