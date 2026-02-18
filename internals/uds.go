package internals

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type SocketType string

const (
	UDSSocket = "unix"
	TCPSocket = "tcp"
)

const UDSName = "unix_socket"

type Socket struct {
	socketType SocketType
	address    string
	pipeWriter *io.PipeWriter
	pipeReader *io.PipeReader
	Logger     *log.Logger
	Echo       io.Writer
	// sends signal when socket is listening
	ready chan bool
}

func GenerateUDSPath(tempPath string) string {
	return strings.Join([]string{tempPath, UDSName}, "_")
}

func NewSocket(socketType SocketType, forceVar bool) (*Socket, error) {
	// TODO: add tcp forked path
	tempPath, err := GetPathCurDir()
	if err != nil {
		return nil, err
	}
	socketPath := GenerateUDSPath(tempPath)
	ok, err := Exists(socketPath)
	if err != nil {
		return nil, err
	}

	if ok {
		if !forceVar {
			return nil, errors.New("socket already exists at this address")
		}
		err := os.Remove(socketPath)
		if err != nil {
			return nil, err
		}
	}
	pipeReader, pipeWriter := io.Pipe()
	newSocket := &Socket{
		socketType: socketType,
		address:    socketPath,
		pipeWriter: pipeWriter,
		pipeReader: pipeReader,
		ready:      make(chan bool),
	}

	return newSocket, nil
}

func (s *Socket) ConnectReader(reader io.Reader) error {
	conn, err := net.Dial(string(s.socketType), s.address)
	s.Logger.Print("input connected")
	if err != nil {
		return err
	}
	defer conn.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		_, err := conn.Write([]byte(line + "\n"))
		if err != nil {
			s.Logger.Printf("err: %s", err)
		}
	}

	return scanner.Err()
}

func (s *Socket) Listen(readers ...io.Reader) error {
	socket, err := net.Listen(string(s.socketType), s.address)
	s.Logger.Print("sockt is listening")
	s.ready <- true
	if err != nil {
		return err
	}

	for {
		// Accept incoming connection
		conn, err := socket.Accept()
		if err != nil {
			s.Logger.Printf("err: couldn't accept connection, %s", err)
		}
		s.Logger.Printf("New connection accepted, %s", conn.RemoteAddr().String())

		// Handle the connection in a separate goroutine
		go func(conn net.Conn) {
			defer conn.Close()
			// buffer for incoming data
			buf := make([]byte, 4096)

			// read data from connection
			for {
				n, err := conn.Read(buf)
				if err != nil {
					s.Logger.Printf("err: couldn't read data, %s", err)
					return
				}

				if n > 0 {
					s.Logger.Print("read from input")

					// let's also pipe to stdout to show the command; we are assuming this is not the terminal stdin
					if s.Echo != nil {
						_, err := s.Echo.Write(buf[:n])
						if err != nil {
							s.Logger.Printf("err: couldn't echo data, %s", err)
						}
					}
					_, err = s.pipeWriter.Write(buf[:n])

					if err != nil {
						s.Logger.Printf("err: couldn't write data, %s", err)
						return
					}
					s.Logger.Print("written to output")

				}
			}
		}(conn)
	}
}

func (s *Socket) Read(p []byte) (int, error) {
	n, err := s.pipeReader.Read(p)

	return n, err
}

func (s *Socket) Close() error {
	return s.pipeWriter.Close()
}

func (s *Socket) CleanUp() error {
	s.Close()
	err := os.Remove(s.address)
	return err
}

func (s *Socket) IsReady() bool {
	return <-s.ready
}
