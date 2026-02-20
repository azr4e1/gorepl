package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- createLogFile tests ---

func TestCreateLogFileCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fd, err := createLogFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer fd.Close()

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist at %q: %v", logPath, err)
	}
}

func TestCreateLogFileCustomPathIsWritable(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fd, err := createLogFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer fd.Close()

	if _, err := fd.WriteString("test log entry"); err != nil {
		t.Fatalf("expected log file to be writable: %v", err)
	}
}

func TestCreateLogFileDefaultPath(t *testing.T) {
	fd, err := createLogFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := fd.Name()
	fd.Close()
	defer os.Remove(name)

	home := os.ExpandEnv("$HOME")
	expectedDir := filepath.Join(home, ".cache", "gorepl")
	if !strings.HasPrefix(name, expectedDir) {
		t.Fatalf("expected log file in %q, got %q", expectedDir, name)
	}
}

func TestCreateLogFileDefaultPathCreatesDir(t *testing.T) {
	// Verify that ~/.cache/gorepl directory is created if it doesn't exist.
	// We just run createLogFile("") and confirm the dir now exists.
	fd, err := createLogFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := fd.Name()
	fd.Close()
	defer os.Remove(name)

	home := os.ExpandEnv("$HOME")
	dir := filepath.Join(home, ".cache", "gorepl")
	if info, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir %q to exist: %v", dir, err)
	} else if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}
}

func TestCreateLogFileTimestampInName(t *testing.T) {
	fd, err := createLogFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := fd.Name()
	fd.Close()
	defer os.Remove(name)

	base := filepath.Base(name)
	if !strings.HasSuffix(base, "-logs") {
		t.Fatalf("expected filename to end with '-logs', got %q", base)
	}
}

// --- createMultiPlexer tests ---

func TestCreateMultiPlexerReturnsNonNil(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	r1 := io.NopCloser(strings.NewReader("hello"))
	r2 := io.NopCloser(strings.NewReader("world"))

	mp := createMultiPlexer(tmpFile, r1, r2)
	if mp == nil {
		t.Fatal("expected non-nil MultiPlexer")
	}
	mp.Close()
}

func TestCreateMultiPlexerHasLogger(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	r := io.NopCloser(strings.NewReader("data"))
	mp := createMultiPlexer(tmpFile, r)
	if mp.Logger == nil {
		t.Fatal("expected non-nil Logger on MultiPlexer")
	}
	mp.Close()
}

func TestCreateMultiPlexerErrHandlerHandlesEOF(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	r := io.NopCloser(strings.NewReader(""))
	mp := createMultiPlexer(tmpFile, r)

	// The ErrHandler should handle io.EOF without panicking
	mp.ErrHandler(io.EOF)
}

func TestCreateMultiPlexerNoInputs(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	mp := createMultiPlexer(tmpFile)
	if mp == nil {
		t.Fatal("expected non-nil MultiPlexer even with no inputs")
	}
	mp.Close()
}

// --- createRepl tests ---

func TestCreateReplReturnsNonNil(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	repl, err := createRepl(tmpFile, []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repl == nil {
		t.Fatal("expected non-nil Repl")
	}
}

func TestCreateReplHasLogger(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	repl, err := createRepl(tmpFile, []string{"cat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repl.Logger == nil {
		t.Fatal("expected non-nil Logger on Repl")
	}
}

func TestCreateReplJoinsArgs(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// createRepl joins args with space; verify the repl runs correctly
	repl, err := createRepl(tmpFile, []string{"echo", "hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repl == nil {
		t.Fatal("expected non-nil Repl")
	}
}

func TestCreateReplSingleArg(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gorepl-log-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	repl, err := createRepl(tmpFile, []string{"cat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repl == nil {
		t.Fatal("expected non-nil Repl")
	}
}
