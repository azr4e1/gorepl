package internals

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuffer is a thread-safe bytes.Buffer for use in tests where goroutines
// write concurrently (e.g. Repl.Run's stdout/stderr goroutines).
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// eventuallyContains polls buf until it contains the expected string or times out.
func eventuallyContains(t *testing.T, buf *safeBuffer, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in output; got: %q", want, buf.String())
}

func TestNewRepl(t *testing.T) {
	r, err := NewRepl("echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Repl")
	}
	if r.cmd != "echo hello" {
		t.Fatalf("expected cmd 'echo hello', got %q", r.cmd)
	}
	if r.Logger == nil {
		t.Fatal("expected non-nil Logger")
	}
}

func TestNewReplEmptyCommand(t *testing.T) {
	r, err := NewRepl("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Repl")
	}
}

func TestReplGetOutputCopiesData(t *testing.T) {
	r, _ := NewRepl("echo hello")
	reader := strings.NewReader("hello world\n")
	var writer bytes.Buffer

	err := r.getOutput("test", reader, &writer)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if writer.String() != "hello world\n" {
		t.Fatalf("expected 'hello world\\n', got %q", writer.String())
	}
}

func TestReplGetOutputMultipleChunks(t *testing.T) {
	r, _ := NewRepl("echo")
	// Create data larger than BufSize to test multiple reads
	data := strings.Repeat("x", BufSize*2+100)
	reader := strings.NewReader(data)
	var writer bytes.Buffer

	err := r.getOutput("test", reader, &writer)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if writer.String() != data {
		t.Fatalf("output mismatch: expected %d bytes, got %d bytes", len(data), writer.Len())
	}
}

func TestReplGetOutputEmpty(t *testing.T) {
	r, _ := NewRepl("echo")
	reader := strings.NewReader("")
	var writer bytes.Buffer

	err := r.getOutput("test", reader, &writer)
	if err != io.EOF {
		t.Fatalf("expected io.EOF on empty reader, got %v", err)
	}
	if writer.Len() != 0 {
		t.Fatalf("expected empty output, got %q", writer.String())
	}
}

func TestReplSendReplStdOut(t *testing.T) {
	r, _ := NewRepl("echo hello")
	reader := strings.NewReader("stdout data\n")
	var writer bytes.Buffer

	err := r.SendReplStdOut(&writer, reader)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if writer.String() != "stdout data\n" {
		t.Fatalf("expected 'stdout data\\n', got %q", writer.String())
	}
}

func TestReplSendReplStdErr(t *testing.T) {
	r, _ := NewRepl("echo hello")
	reader := strings.NewReader("stderr data\n")
	var writer bytes.Buffer

	err := r.SendReplStdErr(&writer, reader)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if writer.String() != "stderr data\n" {
		t.Fatalf("expected 'stderr data\\n', got %q", writer.String())
	}
}

func TestReplSendReplStdinLines(t *testing.T) {
	r, _ := NewRepl("cat")
	input := strings.NewReader("line1\nline2\nline3\n")

	pr, pw := io.Pipe()
	var writer bytes.Buffer

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(&writer, pr)
	}()

	err := r.SendReplStdin(input, pw)
	pw.Close()
	<-done
	pr.Close()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := writer.String()
	if !strings.Contains(got, "line1\n") {
		t.Fatalf("expected 'line1' in output, got %q", got)
	}
	if !strings.Contains(got, "line2\n") {
		t.Fatalf("expected 'line2' in output, got %q", got)
	}
	if !strings.Contains(got, "line3\n") {
		t.Fatalf("expected 'line3' in output, got %q", got)
	}
}

func TestReplSendReplStdinEmpty(t *testing.T) {
	r, _ := NewRepl("cat")
	input := strings.NewReader("")

	pr, pw := io.Pipe()
	var writer bytes.Buffer

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(&writer, pr)
	}()

	err := r.SendReplStdin(input, pw)
	pw.Close()
	<-done
	pr.Close()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writer.Len() != 0 {
		t.Fatalf("expected empty output, got %q", writer.String())
	}
}

// TestReplRunEcho verifies that Run() completes without error for a simple
// command. Output checking is done via SendReplStdOut unit tests; Run()'s
// concurrent design means output may not be flushed to clientInput before
// cmd.Wait() closes the stdout pipe.
func TestReplRunEcho(t *testing.T) {
	r, err := NewRepl("echo hello")
	if err != nil {
		t.Fatal(err)
	}
	pr, pw := io.Pipe()
	var outBuf, errBuf safeBuffer

	done := make(chan error, 1)
	go func() {
		done <- r.Run(pr, &outBuf, &errBuf)
	}()

	// Close stdin — echo doesn't need input
	pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error for valid command: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to complete")
	}
}

// TestReplRunCatForwardsInput verifies that Run() starts a process, passes
// stdin input to it, and completes cleanly.
func TestReplRunCatForwardsInput(t *testing.T) {
	r, err := NewRepl("cat")
	if err != nil {
		t.Fatal(err)
	}

	pr, pw := io.Pipe()
	var outBuf, errBuf safeBuffer

	done := make(chan error, 1)
	go func() {
		done <- r.Run(pr, &outBuf, &errBuf)
	}()

	pw.Write([]byte("test line\n"))
	pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error for valid command: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to complete")
	}
}

func TestReplRunInvalidCommand(t *testing.T) {
	r, err := NewRepl("nonexistent-command-xyz-12345")
	if err != nil {
		t.Fatal(err)
	}

	pr, pw := io.Pipe()
	defer pw.Close()
	var outBuf, errBuf safeBuffer

	done := make(chan error, 1)
	go func() {
		done <- r.Run(pr, &outBuf, &errBuf)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for invalid command")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

func TestReplRunBadShlexQuoting(t *testing.T) {
	// Unclosed quote causes shlex.Split to fail
	r, err := NewRepl(`echo "unclosed`)
	if err != nil {
		t.Fatal(err)
	}

	pr, pw := io.Pipe()
	defer pw.Close()
	var outBuf, errBuf bytes.Buffer

	err = r.Run(pr, &outBuf, &errBuf)
	if err == nil {
		t.Fatal("expected shlex error for unclosed quote")
	}
}

// TestReplRunMultipleLines verifies that Run() handles multiple stdin lines
// correctly and exits cleanly.
func TestReplRunMultipleLines(t *testing.T) {
	r, err := NewRepl("cat")
	if err != nil {
		t.Fatal(err)
	}

	pr, pw := io.Pipe()
	var outBuf, errBuf safeBuffer

	done := make(chan error, 1)
	go func() {
		done <- r.Run(pr, &outBuf, &errBuf)
	}()

	for _, line := range []string{"alpha\n", "beta\n", "gamma\n"} {
		pw.Write([]byte(line))
	}
	pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

func TestReplRunWithCustomLogger(t *testing.T) {
	var logBuf bytes.Buffer
	r, err := NewRepl("echo logged")
	if err != nil {
		t.Fatal(err)
	}
	r.Logger = NewLogger(&logBuf, "testrepl")

	pr, pw := io.Pipe()
	var outBuf, errBuf safeBuffer

	done := make(chan error, 1)
	go func() {
		done <- r.Run(pr, &outBuf, &errBuf)
	}()

	pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Give logger goroutine time to flush
	time.Sleep(50 * time.Millisecond)
	if logBuf.Len() == 0 {
		t.Fatal("expected log output with custom logger")
	}
}
