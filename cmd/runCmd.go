package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
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

	var err error
	switch connectionVar {
	case "uds":
		err = runUDS(args, forceVar, portVar, logPathVar)
	case "tcp":
		err = runTCP(args, forceVar, portVar, logPathVar)
	case "namedpipe":
		err = runNamedPipe(args, forceVar, logPathVar)
	}
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().VarP(&connectionVar, "connection", "c", "type of connection")
	// completion for connection
	runCmd.RegisterFlagCompletionFunc("connection", connectionCompletion)
	runCmd.Flags().StringVarP(&logPathVar, "log", "l", "", "path to logs (default ~/.cache/gorepl)")
	runCmd.Flags().BoolVarP(&forceVar, "force", "f", false, "force connection and start of repl")
	runCmd.Flags().IntVarP(&portVar, "port", "p", 4501, "port for TCP connection")
}

func runTCP(args []string, forceVar bool, portVar int, logPathVar string) error {
	return runSocket(internals.TCPSocket, "TCP", args, forceVar, portVar, logPathVar)
}

func runUDS(args []string, forceVar bool, portVar int, logPathVar string) error {
	return runSocket(internals.UDSSocket, "UDS", args, forceVar, portVar, logPathVar)
}

func runSocket(socketType internals.SocketType, loggerName string, args []string, forceVar bool, portVar int, logPathVar string) error {

	socket, err := internals.NewSocket(socketType, forceVar, portVar)
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
	socket.Logger = internals.NewLogger(logFd, loggerName)

	go socket.Listen()
	// wait for the socket to be ready
	socket.IsReady()

	syncOutput := internals.NewSyncWriter(os.Stdout)
	socketWithEcho := internals.NewReaderWithEcho(socket, syncOutput)
	mp := createMultiPlexer(logFd, socketWithEcho, os.Stdin)
	defer mp.Close()
	go mp.Listen()

	// create repl
	repl, err := createRepl(logFd, args)
	if err != nil {
		return err
	}
	// start repl
	if err := repl.Run(mp, syncOutput, os.Stderr); err != nil {
		return err
	}
	return nil
}

func runNamedPipe(args []string, forceVar bool, logPathVar string) error {
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
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(cacheDir, "gorepl")
	if ok, _ := internals.Exists(logDir); !ok {
		err := os.MkdirAll(logDir, 0777)
		if err != nil {
			return nil, err
		}
	}
	logPath = fmt.Sprintf(filepath.Join(logDir, "%s-logs"), logTime)
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
				mpLogger.Print("error: ", err)
			}
			return
		}
		// check if pipe was closed
		var pipeError *fs.PathError
		if errors.As(err, &pipeError) {
			mpLogger.Print("error: ", err)
			err = mp.Close()
			if err != nil {
				mpLogger.Print("error: ", err)
			}
			return
		}
		mpLogger.Print("error: ", err)
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
	replErr := func(err error) { replLogger.Print("error: ", err) }
	repl.ErrHandler = replErr
	repl.Logger = replLogger

	return repl, nil
}

func sigIntCleanup(cleanup func() error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		fmt.Fprintln(os.Stderr, "\nprocess interrupted, proceeding to cleanup")
		cleanup()
		os.Exit(1)
	}()
}
