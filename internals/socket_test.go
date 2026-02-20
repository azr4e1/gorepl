package internals

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// freePort returns a free TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("could not find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// cleanupSocketPaths removes any leftover socket/trace files before a test.
func cleanupSocketPaths(t *testing.T) {
	t.Helper()
	if uds, err := GenerateUDSPath(); err == nil {
		os.Remove(uds)
	}
	if tcp, err := GenerateTCPPath(); err == nil {
		os.Remove(tcp)
	}
}

// --- Path generation tests ---

func TestGenerateUDSPath(t *testing.T) {
	p, err := GenerateUDSPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, "_"+UDSName) {
		t.Fatalf("expected path to end with '_%s', got %q", UDSName, p)
	}
	if !strings.HasPrefix(p, os.TempDir()) {
		t.Fatalf("expected path to be in temp dir %q, got %q", os.TempDir(), p)
	}
}

func TestGenerateUDSPathDeterministic(t *testing.T) {
	p1, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	p2, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("expected deterministic path, got %q and %q", p1, p2)
	}
}

func TestGenerateTCPPath(t *testing.T) {
	p, err := GenerateTCPPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, "_"+TCPName) {
		t.Fatalf("expected path to end with '_%s', got %q", TCPName, p)
	}
}

func TestGenerateTCPPathDeterministic(t *testing.T) {
	p1, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	p2, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("expected deterministic path, got %q and %q", p1, p2)
	}
}

// --- RetrieveTCPAddress tests ---

func TestRetrieveTCPAddressNotFound(t *testing.T) {
	cleanupSocketPaths(t)
	_, err := RetrieveTCPAddress()
	if err == nil {
		t.Fatal("expected error when TCP address file doesn't exist")
	}
}

func TestRetrieveTCPAddressFound(t *testing.T) {
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	expected := "localhost:9999"
	if err := os.WriteFile(tcpPath, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	addr, err := RetrieveTCPAddress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != expected {
		t.Fatalf("expected %q, got %q", expected, addr)
	}
}

// --- NewSocket tests ---

func TestNewSocketUDS(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil socket")
	}
	s.Close()
}

func TestNewSocketUDSAlreadyExistsNoForce(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	// Create a placeholder file at the socket path
	f, err := os.Create(udsPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(udsPath)

	_, err = NewSocket(UDSSocket, false, 0)
	if err == nil {
		t.Fatal("expected error when socket already exists without force")
	}
}

func TestNewSocketUDSForce(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	// Create a placeholder file
	f, err := os.Create(udsPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, true, 0)
	if err != nil {
		t.Fatalf("unexpected error with force: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil socket")
	}
	s.Close()
}

func TestNewSocketTCP(t *testing.T) {
	cleanupSocketPaths(t)
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	port := freePort(t)
	s, err := NewSocket(TCPSocket, false, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil socket")
	}
	s.Close()
}

func TestNewSocketTCPAlreadyExistsNoForce(t *testing.T) {
	cleanupSocketPaths(t)
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	port := freePort(t)

	// First creation succeeds
	s1, err := NewSocket(TCPSocket, false, port)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()

	// Second creation without force should fail
	_, err = NewSocket(TCPSocket, false, port)
	if err == nil {
		t.Fatal("expected error when TCP trace file already exists without force")
	}
}

func TestNewSocketTCPForce(t *testing.T) {
	cleanupSocketPaths(t)
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	port := freePort(t)

	// First creation
	s1, err := NewSocket(TCPSocket, false, port)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()

	// Second creation with force should succeed
	s2, err := NewSocket(TCPSocket, true, port)
	if err != nil {
		t.Fatalf("unexpected error with force: %v", err)
	}
	if s2 == nil {
		t.Fatal("expected non-nil socket")
	}
	s2.Close()
}

// --- Socket.Close tests ---

func TestSocketClose(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}
}

func TestSocketCloseUnblocksRead(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 10)
		_, err := s.Read(buf)
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	s.Close()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after Close, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not unblock Read")
	}
}

// --- Socket.Listen + send data integration tests ---

func TestSocketUDSListenAndRead(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}

	listenDone := make(chan error, 1)
	go func() {
		listenDone <- s.Listen()
	}()

	if !s.IsReady() {
		t.Fatal("socket not ready")
	}

	// Connect as client and send data
	msg := "hello uds socket\n"
	conn, err := net.Dial("unix", udsPath)
	if err != nil {
		t.Fatalf("couldn't connect: %v", err)
	}
	_, err = conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	conn.Close()

	// Read from socket (via internal pipe)
	readDone := make(chan string, 1)
	go func() {
		buf := make([]byte, len(msg))
		io.ReadFull(s, buf)
		readDone <- string(buf)
	}()

	select {
	case got := <-readDone:
		if got != msg {
			t.Fatalf("expected %q, got %q", msg, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading from UDS socket")
	}

	s.CleanUp()
	<-listenDone
}

func TestSocketTCPListenAndRead(t *testing.T) {
	cleanupSocketPaths(t)
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	port := freePort(t)
	s, err := NewSocket(TCPSocket, false, port)
	if err != nil {
		t.Fatal(err)
	}

	listenDone := make(chan error, 1)
	go func() {
		listenDone <- s.Listen()
	}()

	if !s.IsReady() {
		t.Fatal("socket not ready")
	}

	msg := "hello tcp socket\n"
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("couldn't connect: %v", err)
	}
	_, err = conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	conn.Close()

	readDone := make(chan string, 1)
	go func() {
		buf := make([]byte, len(msg))
		io.ReadFull(s, buf)
		readDone <- string(buf)
	}()

	select {
	case got := <-readDone:
		if got != msg {
			t.Fatalf("expected %q, got %q", msg, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading from TCP socket")
	}

	s.CleanUp()
	<-listenDone
}

func TestSocketUDSMultipleConnections(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}

	go s.Listen()
	if !s.IsReady() {
		t.Fatal("socket not ready")
	}

	const numConns = 3
	msgs := []string{"first\n", "second\n", "third\n"}
	for _, msg := range msgs {
		conn, err := net.Dial("unix", udsPath)
		if err != nil {
			t.Fatalf("couldn't connect: %v", err)
		}
		conn.Write([]byte(msg))
		conn.Close()
	}

	// Collect all data
	resultCh := make(chan string, 1)
	go func() {
		var buf strings.Builder
		b := make([]byte, 256)
		for {
			n, err := s.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if buf.Len() >= len("first\n")+len("second\n")+len("third\n") {
				break
			}
			if err != nil {
				break
			}
		}
		resultCh <- buf.String()
	}()

	select {
	case out := <-resultCh:
		for _, msg := range msgs {
			if !strings.Contains(out, strings.TrimSuffix(msg, "\n")) {
				t.Fatalf("expected %q in output, got %q", msg, out)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out reading from multiple connections")
	}

	s.CleanUp()
}

// --- Socket.CleanUp tests ---

func TestSocketUDSCleanUp(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Listen to create the socket file on disk
	go s.Listen()
	if !s.IsReady() {
		t.Fatal("socket not ready")
	}

	// CleanUp closes the listener. Go's net package removes the UDS file when
	// the listener is closed, so CleanUp's subsequent os.Remove may return
	// "not exist" — that's acceptable; the important thing is the file is gone.
	_ = s.CleanUp()

	// Regardless of which code path removed it, the socket file must be gone.
	ok, err := Exists(udsPath)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected socket file %q to be removed after CleanUp", udsPath)
	}
}

func TestSocketTCPCleanUpRemovesTraceFile(t *testing.T) {
	cleanupSocketPaths(t)
	tcpPath, err := GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}

	port := freePort(t)
	s, err := NewSocket(TCPSocket, false, port)
	if err != nil {
		t.Fatal(err)
	}

	// Trace file should exist now
	ok, err := Exists(tcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected TCP trace file to exist after NewSocket")
	}

	_ = s.CleanUp()

	// Trace file should be removed
	ok, err = Exists(tcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected TCP trace file to be removed after CleanUp")
	}
}

// --- Socket.IsReady tests ---

func TestSocketIsReady(t *testing.T) {
	cleanupSocketPaths(t)
	udsPath, err := GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(udsPath)

	s, err := NewSocket(UDSSocket, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ready := make(chan bool, 1)
	go func() {
		go s.Listen()
		ready <- s.IsReady()
	}()

	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("expected IsReady to return true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for IsReady")
	}
}
