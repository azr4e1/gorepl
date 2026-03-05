package cmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
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
		err = sendNamedPipe(command, args, addressVarSend)
	case "uds":
		err = sendUDS(command, args, addressVarSend)
	case "tcp":
		err = sendTCP(command, args, portVarSend)
	}
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().VarP(&connectionVarSend, "connection", "c", "type of connection")
	// add completion for connection
	sendCmd.RegisterFlagCompletionFunc("connection", connectionCompletion)
	sendCmd.Flags().IntVarP(&portVarSend, "port", "p", -1, "port for TCP connection")
	// add completion for port
	sendCmd.RegisterFlagCompletionFunc("port", portCompletion)
	sendCmd.Flags().StringVarP(&addressVarSend, "address", "a", "", "address for namedpipe or UDS connection")
	// completion for address
	sendCmd.RegisterFlagCompletionFunc("address", addressCompletion)
}

func sendTCP(command *cobra.Command, args []string, portVarSend int) error {
	socketType := internals.TCPSocket
	var address string
	var err error
	if portVarSend >= 0 {
		address = net.JoinHostPort(internals.TCPHost, strconv.Itoa(portVarSend))
	} else {
		address, err = internals.RetrieveTCPAddress()
	}
	if err != nil {
		return err
	}

	return sendSocket(command, args, socketType, address)
}

func sendUDS(command *cobra.Command, args []string, addressVarSend string) error {
	socketType := internals.UDSSocket
	var address string
	var err error
	if addressVarSend != "" {
		address = addressVarSend
	} else {
		address, err = internals.GenerateUDSPath()
	}
	if err != nil {
		return err
	}

	return sendSocket(command, args, socketType, address)
}

func sendSocket(command *cobra.Command, args []string, socketType internals.SocketType, address string) error {
	conn, err := net.Dial(string(socketType), address)
	if err != nil {
		return fmt.Errorf("couldn't connect to a %s socket. Are you sure your repl is running at this address?", socketType)
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

func sendNamedPipe(command *cobra.Command, args []string, addressVarSend string) error {
	// get connection
	var nPipe *internals.TempNPipe
	var err error
	if addressVarSend != "" {
		nPipe, err = internals.NewFifo(addressVarSend)
	} else {
		nPipe, err = internals.NewTempFifo()
	}
	if err != nil {
		return errors.New("couldn't connect to a named pipe. Are you sure your repl is running at this address?")
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
