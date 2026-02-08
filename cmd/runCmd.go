package cmd

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the shell",
		Run:   Run,
		Args:  cobra.MaximumNArgs(1),
	}
)

func Run(command *cobra.Command, args []string) {
	if len(args) == 0 {
		command.Help()
		os.Exit(0)
	}
	// TODO: create directory specific pipe
	pipe, err := internals.MkTempFifo("test")
	if err != nil {
		cobra.CheckErr(err)
	}

	logFd, err := os.Create("logs")
	if err != nil {
		cobra.CheckErr(err)
	}
	cmd := strings.Join(args, " ")

	repl, err := internals.NewRepl(cmd)
	if err != nil {
		cobra.CheckErr(err)
	}
	replLogger := internals.NewLogger(logFd, "Repl")
	replErr := func(err error) { replLogger.Print("err: ", err) }
	repl.ErrHandler = replErr
	repl.Logger = replLogger

	syncOutput := internals.NewSyncWriter(os.Stdout)
	mp := internals.NewMultiPlexer([]io.Reader{pipe, os.Stdin}, []io.Writer{syncOutput})
	defer mp.Close()
	mpLogger := internals.NewLogger(logFd, "MultiPlexer")
	mpErr := func(err error) {
		if errors.Is(err, io.EOF) {
			mp.Close()
			return
		}
		// check if pipe was closed
		var pipeError *fs.PathError
		if errors.As(err, &pipeError) {
			return
		}
		mpLogger.Printf("err: %T, %s", err, err)
		// mpLogger.Print("err: ", err)
	}
	mp.Logger = mpLogger
	mp.ErrHandler = mpErr

	var wg sync.WaitGroup
	wg.Go(mp.Listen)
	if err := repl.Run(mp, syncOutput, os.Stderr); err != nil {
		cobra.CheckErr(err)
	}
	pipe.Close()
	wg.Wait()
}

func init() {
	rootCmd.AddCommand(runCmd)
	// runCmd.Flags().IntVarP()
}
