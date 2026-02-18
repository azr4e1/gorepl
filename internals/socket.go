package internals

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
)

type SocketType string

const (
	UDSSocket SocketType = "unix"
	TCPSocket SocketType = "tcp"
)

const UDSName = "unix_socket"

type Socket struct {
	socketType SocketType
	address    string
	pipeWriter *io.PipeWriter
	pipeReader *io.PipeReader
	Logger     *log.Logger
	// sends signal when socket is ready listening
	ready chan bool
}

func GenerateUDSPath(tempPath string) string {
	return strings.Join([]string{tempPath, UDSName}, "_")
}

func NewSocket(socketType SocketType, forceVar bool, port int) (*Socket, error) {
	var address string
	tempPath, err := GetPathCurDir()
	if err != nil {
		return nil, err
	}
	if socketType == UDSSocket {
		address = GenerateUDSPath(tempPath)
		ok, err := Exists(address)
		if err != nil {
			return nil, err
		}

		if ok {
			if !forceVar {
				return nil, errors.New("socket already exists at this address")
			}
			err := os.Remove(address)
			if err != nil {
				return nil, err
			}
		}
	} else {
		url := fmt.Sprintf("localhost:%d", port)
		address = path.Join(url, tempPath)
	}
	pipeReader, pipeWriter := io.Pipe()
	newSocket := &Socket{
		socketType: socketType,
		address:    address,
		pipeWriter: pipeWriter,
		pipeReader: pipeReader,
		ready:      make(chan bool),
		Logger:     DiscardLogger,
	}

	return newSocket, nil
}

func (s *Socket) ConnectReader(reader io.Reader) error {
	conn, err := net.Dial(string(s.socketType), s.address)
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

	// close the pipeWriter to signal that input is closed
	if err := scanner.Err(); err != nil {
		s.Logger.Printf("err: %s", err)
	}
	s.Logger.Printf("closing input")
	return s.pipeWriter.Close()
}

func (s *Socket) Listen() error {
	socket, err := net.Listen(string(s.socketType), s.address)
	if err != nil {
		return err
	}
	defer socket.Close()
	s.Logger.Printf("%s socket is listening", s.socketType)
	s.ready <- true

	// TODO: need to find an exit path
	for {
		// Accept incoming connection
		conn, err := socket.Accept()
		if err != nil {
			s.Logger.Printf("err: couldn't accept connection, %s", err)
			continue
		}
		s.Logger.Print("New connection accepted")

		// Handle the connection in a separate goroutine
		go s.readFromConnection(conn)
	}
}

func (s *Socket) readFromConnection(conn net.Conn) {
	defer conn.Close()
	// buffer for incoming data
	buf := make([]byte, 4096)

	// read data from connection
	for {
		n, err := conn.Read(buf)
		if err != nil {
			s.Logger.Printf("err: couldn't read data, %s", err)
		}

		if n > 0 {
			s.Logger.Print("read from input")

			_, err = s.pipeWriter.Write(buf[:n])

			if err != nil {
				s.Logger.Printf("err: couldn't write data, %s", err)
			}
			s.Logger.Print("written to output")
		}
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
