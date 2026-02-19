package cmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

var zeroArgsError = errors.New("no arguments provided")

var (
	sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send data to repl",
		Run:   Send,
	}
)

func Send(command *cobra.Command, args []string) {
	var err error
	switch connectionVarSend {
	case "namedpipe":
		err = sendNamedPipe(command, args)
	case "uds":
		err = sendUDS(command, args)
	case "tcp":
		err = sendTCP(command, args)
	}
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().VarP(&connectionVarSend, "connection", "c", "type of connection")
	sendCmd.Flags().IntVarP(&portVarSend, "port", "p", 4501, "port for TCP connection")
}

func sendTCP(command *cobra.Command, args []string) error {
	socketType := internals.TCPSocket
	address := fmt.Sprintf("localhost:%d", portVarSend)

	return sendSocket(command, args, socketType, address)
}

func sendUDS(command *cobra.Command, args []string) error {
	socketType := internals.UDSSocket
	tempPath, err := internals.GetPathCurDir()
	if err != nil {
		return err
	}
	address := internals.GenerateUDSPath(tempPath)

	return sendSocket(command, args, socketType, address)
}

func sendSocket(command *cobra.Command, args []string, socketType internals.SocketType, address string) error {
	conn, err := net.Dial(string(socketType), address)
	if err != nil {
		return err
	}
	defer conn.Close()

	lines, err := getContent(args)
	if err != nil {
		if errors.Is(err, zeroArgsError) {
			command.Help()
			return nil
		}
		return err
	}
	_, err = conn.Write(preprocess(lines))
	return err
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
	lines, err := getContent(args)
	if err != nil {
		if errors.Is(err, zeroArgsError) {
			command.Help()
			return nil
		}
		return err
	}
	_, err = nPipe.Write(preprocess(lines))
	return err
}

func getContent(args []string) ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	var lines []byte
	// we are getting piped
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		lines, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
	} else {
		if len(args) == 0 {
			return nil, zeroArgsError
		}
		lines = fmt.Appendln(nil, strings.Join(args, " "))
	}

	return lines, nil
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
