package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"

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
	tempDirPath, err := internals.GetNPipePathCurDir()
	if err != nil {
		cobra.CheckErr(err)
	}
	pipe, err := internals.MkTempFifo(tempDirPath)
	if err != nil {
		cobra.CheckErr(err)
	}

	logTime := time.Now().Format("2006-01-02T15:04:05")
	logPath := os.ExpandEnv("$HOME/.cache/gorepl")
	if ok, _ := internals.Exists(logPath); !ok {
		os.MkdirAll(logPath, 0777)
	}
	logFd, err := os.Create(fmt.Sprintf(path.Join(logPath, "%s-logs"), logTime))
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
	pipeEcho := internals.NewReaderWithEcho(pipe, syncOutput)
	mp := internals.NewMultiPlexer(pipeEcho, os.Stdin)
	mpLogger := internals.NewLogger(logFd, "MultiPlexer")
	mpErr := func(err error) {
		if errors.Is(err, io.EOF) {
			err = mp.Close()
			mpLogger.Print("err: ", err)
			return
		}
		// check if pipe was closed
		var pipeError *fs.PathError
		if errors.As(err, &pipeError) {
			mpLogger.Print("err: ", err)
			err = mp.Close()
			mpLogger.Print("err: ", err)
			return
		}
		mpLogger.Print("err: ", err)
	}
	mp.Logger = mpLogger
	mp.ErrHandler = mpErr

	var wg sync.WaitGroup
	wg.Go(mp.Listen)
	if err := repl.Run(mp, syncOutput, os.Stderr); err != nil {
		cobra.CheckErr(err)
	}
	pipe.Close()
	mp.Close()
	wg.Wait()
}

func init() {
	rootCmd.AddCommand(runCmd)
	// runCmd.Flags().IntVarP()
}
