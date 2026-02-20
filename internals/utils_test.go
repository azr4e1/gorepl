package internals

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// --- syncWriter tests ---

func TestSyncWriterWrite(t *testing.T) {
	var buf bytes.Buffer
	sw := NewSyncWriter(&buf)
	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", buf.String())
	}
}

func TestSyncWriterEmpty(t *testing.T) {
	var buf bytes.Buffer
	sw := NewSyncWriter(&buf)
	n, err := sw.Write([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 bytes, got %d", n)
	}
}

func TestSyncWriterConcurrent(t *testing.T) {
	var buf bytes.Buffer
	sw := NewSyncWriter(&buf)
	var wg sync.WaitGroup
	const goroutines = 100
	for range goroutines {
		wg.Go(func() {
			sw.Write([]byte("x"))
		})
	}
	wg.Wait()
	if buf.Len() != goroutines {
		t.Fatalf("expected %d bytes, got %d", goroutines, buf.Len())
	}
}

// --- NewLogger tests ---

func TestNewLoggerNoName(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "")
	logger.Print("test message")
	out := buf.String()
	if !strings.Contains(out, "test message") {
		t.Fatalf("expected log output to contain 'test message', got %q", out)
	}
	// With no name, prefix should not contain "name:"
	// (time format contains ":" but the prefix itself is just " ")
	if strings.HasPrefix(out, ":") {
		t.Fatalf("prefix should not start with colon, got %q", out)
	}
}

func TestNewLoggerWithName(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "mylogger")
	logger.Print("test message")
	out := buf.String()
	if !strings.Contains(out, "test message") {
		t.Fatalf("expected log output to contain 'test message', got %q", out)
	}
	if !strings.Contains(out, "mylogger:") {
		t.Fatalf("expected log output to contain 'mylogger:', got %q", out)
	}
}

func TestNewLoggerConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "concurrent")
	var wg sync.WaitGroup
	const goroutines = 50
	for range goroutines {
		wg.Go(func() {
			logger.Print("line")
		})
	}
	wg.Wait()
	// Should not panic or deadlock; output should contain multiple "line" entries
	if !strings.Contains(buf.String(), "line") {
		t.Fatal("expected at least one log line")
	}
}

func TestDiscardLogger(t *testing.T) {
	// Should not panic and output should go nowhere
	DiscardLogger.Print("this is discarded")
	DiscardLogger.Printf("formatted %s", "discard")
}

func TestDefaultErrHandler(t *testing.T) {
	// Should not panic
	DefaultErrHandler(io.EOF)
	DefaultErrHandler(nil)
}

// --- ReaderWithEcho tests ---

func TestReaderWithEchoRead(t *testing.T) {
	data := []byte("hello world")
	reader := io.NopCloser(bytes.NewReader(data))
	var echoBuf bytes.Buffer
	rwe := NewReaderWithEcho(reader, &echoBuf)

	buf := make([]byte, len(data))
	n, err := rwe.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if string(buf[:n]) != string(data) {
		t.Fatalf("expected %q, got %q", data, buf[:n])
	}
	if echoBuf.String() != string(data) {
		t.Fatalf("echo: expected %q, got %q", data, echoBuf.String())
	}
}

func TestReaderWithEchoEOF(t *testing.T) {
	reader := io.NopCloser(bytes.NewReader([]byte{}))
	var echoBuf bytes.Buffer
	rwe := NewReaderWithEcho(reader, &echoBuf)

	buf := make([]byte, 10)
	_, err := rwe.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	// On EOF, nothing should be echoed
	if echoBuf.Len() != 0 {
		t.Fatalf("expected empty echo buf on EOF, got %q", echoBuf.String())
	}
}

func TestReaderWithEchoPartialRead(t *testing.T) {
	data := []byte("abcdefghij")
	reader := io.NopCloser(bytes.NewReader(data))
	var echoBuf bytes.Buffer
	rwe := NewReaderWithEcho(reader, &echoBuf)

	// Read only 5 bytes at a time
	buf := make([]byte, 5)
	n, err := rwe.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Fatal("expected to read some bytes")
	}
	// Echo should contain exactly the bytes read
	if echoBuf.String() != string(data[:n]) {
		t.Fatalf("echo: expected %q, got %q", data[:n], echoBuf.String())
	}
}

func TestReaderWithEchoClose(t *testing.T) {
	reader := io.NopCloser(bytes.NewReader([]byte("test")))
	var echoBuf bytes.Buffer
	rwe := NewReaderWithEcho(reader, &echoBuf)
	if err := rwe.Close(); err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}
}

// --- Exists tests ---

func TestExistsTrue(t *testing.T) {
	f, err := os.CreateTemp("", "gorepl-test-exists-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	ok, err := Exists(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected file to exist")
	}
}

func TestExistsFalse(t *testing.T) {
	ok, err := Exists("/tmp/gorepl-test-nonexistent-file-xyzzy-99999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected file to not exist")
	}
}

func TestExistsDirectory(t *testing.T) {
	dir, err := os.MkdirTemp("", "gorepl-test-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	ok, err := Exists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected directory to exist")
	}
}

// --- GetPathCurDir tests ---

func TestGetPathCurDirNonEmpty(t *testing.T) {
	p, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestGetPathCurDirDeterministic(t *testing.T) {
	p1, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p2, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1 != p2 {
		t.Fatalf("expected same path on repeated calls, got %q and %q", p1, p2)
	}
}

func TestGetPathCurDirInTempDir(t *testing.T) {
	p, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tmpDir := os.TempDir()
	if !strings.HasPrefix(p, tmpDir) {
		t.Fatalf("expected path to start with temp dir %q, got %q", tmpDir, p)
	}
}

func TestGetPathCurDirContainsGorepl(t *testing.T) {
	p, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(p, "gorepl_") {
		t.Fatalf("expected path to contain 'gorepl_', got %q", p)
	}
}

func TestGetPathCurDirDifferentDirs(t *testing.T) {
	p1, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	tmpDir, err := os.MkdirTemp("", "gorepl-test-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	p2, err := GetPathCurDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1 == p2 {
		t.Fatal("expected different paths for different directories")
	}
}
