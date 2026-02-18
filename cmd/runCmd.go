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
	} else if udsVar {
		err := runUDS(args)
		if err != nil {
			cobra.CheckErr(err)
		}
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&namedPipeVar, "named-pipe", "n", false, "Connect to a named pipe")
	runCmd.Flags().BoolVarP(&udsVar, "uds", "u", true, "Connect to a unix domain socket")
	runCmd.Flags().StringVarP(&logPathVar, "log", "l", "", "path to logs; defaults to ~/.cache/gorepl")
	runCmd.Flags().BoolVarP(&forceVar, "force", "f", false, "Force connection and start of repl")
}

func runUDS(args []string) error {

	fmt.Println("UDS launched")
	socket, err := internals.NewSocket(internals.UDSSocket, forceVar)
	if err != nil {
		return err
	}
	defer socket.CleanUp()

	sigIntCleanup(func() error { return socket.CleanUp() })

	// create loggers
	logFd, err := createLogFile(logPathVar)
	if err != nil {
		return err
	}
	socket.Logger = internals.NewLogger(logFd, "UDS")
	// create echo stdin
	syncOutput := internals.NewSyncWriter(os.Stdout)
	socket.Echo = syncOutput

	go socket.Listen()
	// wait for the socket to be read
	socket.IsReady()

	go socket.ConnectReader(os.Stdin)
	socket.Logger.Print("Stdin connected")
	// pipeEcho := internals.NewReaderWithEcho(pipe, syncOutput)

	// create repl
	repl, err := createRepl(logFd, args)
	if err != nil {
		return err
	}
	// start repl
	if err := repl.Run(socket, syncOutput, os.Stderr); err != nil {
		return err
	}
	return nil
}

func runNamedPipe(args []string) error {
	// connect to pipe
	pipe, err := internals.MkTempFifo(forceVar)
	if err != nil {
		return err
	}
	// Cleanup when exiting normally
	defer pipe.CleanUp()

	// handle process interruption
	sigIntCleanup(pipe.CleanUp)

	// create loggers
	logFd, err := createLogFile(logPathVar)
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

func createLogFile(logPath string) (*os.File, error) {

	if logPath != "" {
		return os.Create(logPath)
	}

	logTime := time.Now().Format("2006-01-02T15:04:05")
	logDir := os.ExpandEnv("$HOME/.cache/gorepl")
	if ok, _ := internals.Exists(logDir); !ok {
		err := os.MkdirAll(logDir, 0777)
		if err != nil {
			return nil, err
		}
	}
	logPath = fmt.Sprintf(path.Join(logPath, "%s-logs"), logTime)
	return os.Create(logPath)
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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		fmt.Fprintln(os.Stderr, "\nProcess interrupted, proceeding to cleanup")
		cleanup()
		os.Exit(1)
	}()
}
