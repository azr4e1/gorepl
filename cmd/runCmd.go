package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the shell",
		Run:   Run,
	}
)

func Run(command *cobra.Command, args []string) {
	if len(args) == 0 {
		command.Help()
		os.Exit(0)
	}

	if namedPipeVar {
		err := runNamedPipe(args)
		if err != nil {
			cobra.CheckErr(err)
		}
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&namedPipeVar, "named-pipe", "n", true, "Connect to a named pipe")
	runCmd.Flags().BoolVarP(&forceVar, "force", "f", false, "Force connection and start of repl")
}

func runNamedPipe(args []string) error {
	// connect to pipe
	tempDirPath, err := internals.GetNPipePathCurDir()
	if err != nil {
		return err
	}

	pipe, err := internals.MkTempFifo(tempDirPath, forceVar)
	if err != nil {
		return err
	}
	// Cleanup when exiting normally
	defer pipe.CleanUp()

	// handle process interruption
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		fmt.Fprintln(os.Stderr, "Process interrupted, proceeding to cleanup")
		pipe.CleanUp()
		os.Exit(1)
	}()

	// create loggers
	logTime := time.Now().Format("2006-01-02T15:04:05")
	logPath := os.ExpandEnv("$HOME/.cache/gorepl")
	if ok, _ := internals.Exists(logPath); !ok {
		err := os.MkdirAll(logPath, 0777)
		if err != nil {
			return err
		}
	}
	logFd, err := os.Create(fmt.Sprintf(path.Join(logPath, "%s-logs"), logTime))
	if err != nil {
		return err
	}
	cmd := strings.Join(args, " ")

	// create repl
	repl, err := internals.NewRepl(cmd)
	if err != nil {
		return err
	}
	replLogger := internals.NewLogger(logFd, "Repl")
	replErr := func(err error) { replLogger.Print("Error: ", err) }
	repl.ErrHandler = replErr
	repl.Logger = replLogger

	// create echo stdin
	syncOutput := internals.NewSyncWriter(os.Stdout)
	pipeEcho := internals.NewReaderWithEcho(pipe, syncOutput)

	// create multiplexer
	mp := internals.NewMultiPlexer(pipeEcho, os.Stdin)
	mpLogger := internals.NewLogger(logFd, "MultiPlexer")
	mpErr := func(err error) {
		// check if any input has reached EOF
		if errors.Is(err, io.EOF) {
			err = mp.Close()
			if err != nil {
				mpLogger.Print("Error: ", err)
			}
			return
		}
		// check if pipe was closed
		var pipeError *fs.PathError
		if errors.As(err, &pipeError) {
			mpLogger.Print("Error: ", err)
			err = mp.Close()
			if err != nil {
				mpLogger.Print("Error: ", err)
			}
			return
		}
		mpLogger.Print("Error: ", err)
	}
	mp.Logger = mpLogger
	mp.ErrHandler = mpErr

	// listen for incoming data
	go mp.Listen()
	// start repl
	if err := repl.Run(mp, syncOutput, os.Stderr); err != nil {
		return err
	}
	// close pipes
	mp.Close()

	return nil
}
