package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
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

func runUDS(args []string) error {
	tempPath, err := internals.GetPathCurDir()
	if err != nil {
		return err
	}
	socketPath := internals.GenerateUDSPath(tempPath)
	ok, err := internals.Exists(socketPath)
	if err != nil {
		return err
	}

	if ok {
		if !forceVar {
			return errors.New("pipe already exists at this address")
		}
		err := os.Remove(socketPath)
		if err != nil {
			return err
		}
	}

	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		os.Remove(socketPath)
		return err
	}

	sigIntCleanup(func() error { return os.Remove(socketPath) })

	// create loggers
	logFd, err := createLogFile()
	if err != nil {
		return err
	}

	// create repl
	repl, err := createRepl(logFd, args)
	if err != nil {
		return err
	}

}

func runNamedPipe(args []string) error {
	// connect to pipe
	tempPath, err := internals.GetPathCurDir()
	if err != nil {
		return err
	}

	pipe, err := internals.MkTempFifo(tempPath, forceVar)
	if err != nil {
		return err
	}
	// Cleanup when exiting normally
	defer pipe.CleanUp()

	// handle process interruption
	sigIntCleanup(pipe.CleanUp)

	// create loggers
	logFd, err := createLogFile()
	if err != nil {
		return err
	}

	// create repl
	repl, err := createRepl(logFd, args)
	if err != nil {
		return err
	}

	// create echo stdin
	syncOutput := internals.NewSyncWriter(os.Stdout)
	pipeEcho := internals.NewReaderWithEcho(pipe, syncOutput)

	// create multiplexer
	mp := createMultiPlexer(logFd, pipeEcho, os.Stdin)

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

func createLogFile() (*os.File, error) {

	logTime := time.Now().Format("2006-01-02T15:04:05")
	logPath := os.ExpandEnv("$HOME/.cache/gorepl")
	if ok, _ := internals.Exists(logPath); !ok {
		err := os.MkdirAll(logPath, 0777)
		if err != nil {
			return nil, err
		}
	}
	logFd, err := os.Create(fmt.Sprintf(path.Join(logPath, "%s-logs"), logTime))
	if err != nil {
		return nil, err
	}

	return logFd, nil
}

func createMultiPlexer(logFile *os.File, inputs ...io.ReadCloser) *internals.MultiPlexer {
	mp := internals.NewMultiPlexer(inputs...)
	mpLogger := internals.NewLogger(logFile, "MultiPlexer")
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

	return mp
}

func createRepl(logFile *os.File, args []string) (*internals.Repl, error) {
	cmd := strings.Join(args, " ")
	repl, err := internals.NewRepl(cmd)
	if err != nil {
		return nil, err
	}
	replLogger := internals.NewLogger(logFile, "Repl")
	replErr := func(err error) { replLogger.Print("Error: ", err) }
	repl.ErrHandler = replErr
	repl.Logger = replLogger

	return repl, nil
}

func sigIntCleanup(cleanup func() error) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		fmt.Fprintln(os.Stderr, "\nProcess interrupted, proceeding to cleanup")
		cleanup()
		os.Exit(1)
	}()
}
