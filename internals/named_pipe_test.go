package internals

import (
	"os"
	"strings"
	"testing"
	"time"
)

// cleanupNPipePath removes any leftover named pipe file before a test.
func cleanupNPipePath(t *testing.T) {
	t.Helper()
	if p, err := GenerateNPipePath(); err == nil {
		os.Remove(p)
	}
}

// --- GenerateNPipePath tests ---

func TestGenerateNPipePath(t *testing.T) {
	p, err := GenerateNPipePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == "" {
		t.Fatal("expected non-empty path")
	}
	if !strings.HasSuffix(p, "_"+NPipeName) {
		t.Fatalf("expected path to end with '_%s', got %q", NPipeName, p)
	}
}

func TestGenerateNPipePathDeterministic(t *testing.T) {
	p1, err := GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}
	p2, err := GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("expected same path on repeated calls, got %q and %q", p1, p2)
	}
}

func TestGenerateNPipePathInTempDir(t *testing.T) {
	p, err := GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p, os.TempDir()) {
		t.Fatalf("expected path in temp dir %q, got %q", os.TempDir(), p)
	}
}

// --- MkTempFifo tests ---

func TestMkTempFifo(t *testing.T) {
	cleanupNPipePath(t)
	npipePath, err := GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pipe.CleanUp()

	if pipe.Address != npipePath {
		t.Fatalf("expected address %q, got %q", npipePath, pipe.Address)
	}
	ok, err := Exists(pipe.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected named pipe to exist on filesystem after MkTempFifo")
	}
}

func TestMkTempFifoAlreadyExistsNoForce(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	pipe.Close() // close fd but keep the file on disk
	defer os.Remove(pipe.Address)

	// Attempt to create again without force — should fail
	_, err = MkTempFifo(false)
	if err == nil {
		t.Fatal("expected error when pipe already exists and force=false")
	}
}

func TestMkTempFifoForce(t *testing.T) {
	cleanupNPipePath(t)

	pipe1, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	pipe1.Close() // keep the file on disk
	defer os.Remove(pipe1.Address)

	// Force-recreate should succeed
	pipe2, err := MkTempFifo(true)
	if err != nil {
		t.Fatalf("unexpected error with force: %v", err)
	}
	defer pipe2.CleanUp()

	if pipe2.Address != pipe1.Address {
		t.Fatalf("expected same address, got %q and %q", pipe1.Address, pipe2.Address)
	}
}

// --- NewFifo / NewTempFifo tests ---

func TestNewFifoNotExist(t *testing.T) {
	_, err := NewFifo("/tmp/gorepl-test-nonexistent-pipe-xyzzy-99999")
	if err == nil {
		t.Fatal("expected error for non-existent pipe path")
	}
}

func TestNewFifoExists(t *testing.T) {
	cleanupNPipePath(t)

	pipe1, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe1.CleanUp()

	// Open the same path with NewFifo
	pipe2, err := NewFifo(pipe1.Address)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pipe2.Close()

	if pipe2.Address != pipe1.Address {
		t.Fatalf("expected address %q, got %q", pipe1.Address, pipe2.Address)
	}
}

func TestNewTempFifoNotExist(t *testing.T) {
	cleanupNPipePath(t)
	_, err := NewTempFifo()
	if err == nil {
		t.Fatal("expected error when no named pipe exists for current dir")
	}
}

func TestNewTempFifoExists(t *testing.T) {
	cleanupNPipePath(t)

	expected, err := GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}

	pipe1, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe1.CleanUp()

	pipe2, err := NewTempFifo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pipe2.Close()

	if pipe2.Address != expected {
		t.Fatalf("expected address %q, got %q", expected, pipe2.Address)
	}
}

// --- TempNPipe Read / Write tests ---

func TestTempNPipeWriteRead(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe.CleanUp()

	// Named pipe opened O_RDWR — write and read on the same fd.
	// The kernel buffers the data, so we can write then read.
	msg := []byte("hello named pipe\n")
	readDone := make(chan []byte, 1)

	go func() {
		buf := make([]byte, len(msg))
		n, _ := pipe.Read(buf)
		readDone <- buf[:n]
	}()

	// Small delay to let the goroutine block on Read
	time.Sleep(20 * time.Millisecond)

	n, err := pipe.Write(msg)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len(msg) {
		t.Fatalf("expected %d bytes written, got %d", len(msg), n)
	}

	select {
	case got := <-readDone:
		if string(got) != string(msg) {
			t.Fatalf("expected %q, got %q", msg, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading from named pipe")
	}
}

func TestTempNPipeWriteMultipleChunks(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe.CleanUp()

	chunks := []string{"first\n", "second\n", "third\n"}
	total := 0
	for _, c := range chunks {
		total += len(c)
	}

	readDone := make(chan string, 1)
	go func() {
		buf := make([]byte, total)
		n, _ := pipe.Read(buf)
		readDone <- string(buf[:n])
	}()

	time.Sleep(20 * time.Millisecond)

	for _, c := range chunks {
		if _, err := pipe.Write([]byte(c)); err != nil {
			t.Fatalf("write error: %v", err)
		}
	}

	select {
	case got := <-readDone:
		if len(got) == 0 {
			t.Fatal("expected some data read from pipe")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading chunks from named pipe")
	}
}

// --- TempNPipe Close / CleanUp tests ---

func TestTempNPipeClose(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pipe.Address)

	if err := pipe.Close(); err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}
}

func TestTempNPipeCleanUp(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}
	addr := pipe.Address

	if err := pipe.CleanUp(); err != nil {
		t.Fatalf("unexpected error on CleanUp: %v", err)
	}

	ok, err := Exists(addr)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected named pipe to be removed after CleanUp")
	}
}

func TestTempNPipeCleanUpIdempotent(t *testing.T) {
	cleanupNPipePath(t)

	pipe, err := MkTempFifo(false)
	if err != nil {
		t.Fatal(err)
	}

	_ = pipe.CleanUp()
	// Second CleanUp should not panic (best effort — errors are expected)
	_ = pipe.CleanUp()
}
