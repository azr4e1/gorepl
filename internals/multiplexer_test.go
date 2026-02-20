package internals

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewMultiPlexer(t *testing.T) {
	r := io.NopCloser(strings.NewReader("hello"))
	mp := NewMultiPlexer(r)
	if mp == nil {
		t.Fatal("expected non-nil MultiPlexer")
	}
	if mp.pipeReader == nil || mp.pipeWriter == nil {
		t.Fatal("expected non-nil internal pipe")
	}
	if mp.Logger == nil {
		t.Fatal("expected non-nil Logger")
	}
}

func TestNewMultiPlexerNoInputs(t *testing.T) {
	mp := NewMultiPlexer()
	if mp == nil {
		t.Fatal("expected non-nil MultiPlexer")
	}
	// Calling Listen with no inputs should close the pipe immediately.
	done := make(chan struct{})
	go func() {
		mp.Listen()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: Listen with no inputs should return quickly")
	}
}

func TestMultiPlexerReadSingleInput(t *testing.T) {
	data := "hello\n"
	r := io.NopCloser(strings.NewReader(data))
	mp := NewMultiPlexer(r)

	go mp.Listen()

	// Read all bytes using a timeout-guarded goroutine.
	resultCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		tmp := make([]byte, 64)
		for {
			n, err := mp.Read(tmp)
			if n > 0 {
				buf.Write(tmp[:n])
			}
			if err != nil {
				break
			}
		}
		resultCh <- buf.String()
	}()

	select {
	case got := <-resultCh:
		if got != data {
			t.Fatalf("expected %q, got %q", data, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading from multiplexer")
	}
}

func TestMultiPlexerMultipleInputs(t *testing.T) {
	r1 := io.NopCloser(strings.NewReader("aaa"))
	r2 := io.NopCloser(strings.NewReader("bbb"))
	mp := NewMultiPlexer(r1, r2)

	go mp.Listen()

	resultCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		tmp := make([]byte, 64)
		for {
			n, err := mp.Read(tmp)
			if n > 0 {
				buf.Write(tmp[:n])
			}
			if err != nil {
				break
			}
		}
		resultCh <- buf.String()
	}()

	select {
	case out := <-resultCh:
		if !strings.Contains(out, "aaa") || !strings.Contains(out, "bbb") {
			t.Fatalf("expected both 'aaa' and 'bbb' in output, got %q", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading from multiplexer")
	}
}

func TestMultiPlexerEOFAfterAllInputsClosed(t *testing.T) {
	r := io.NopCloser(strings.NewReader(""))
	mp := NewMultiPlexer(r)

	go mp.Listen()

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 10)
		_, err := mp.Read(buf)
		done <- err
	}()

	select {
	case err := <-done:
		if err != io.EOF && err != io.ErrClosedPipe {
			t.Fatalf("expected EOF or ErrClosedPipe after all inputs close, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EOF")
	}
}

func TestMultiPlexerCloseIdempotent(t *testing.T) {
	r := io.NopCloser(strings.NewReader("test"))
	mp := NewMultiPlexer(r)

	// Close multiple times should not panic (sync.Once guarantees single execution)
	_ = mp.Close()
	_ = mp.Close()
	_ = mp.Close()
}

func TestMultiPlexerCloseUnblocksRead(t *testing.T) {
	// A pipe that never sends data
	pr, _ := io.Pipe()
	mp := NewMultiPlexer(pr)

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 10)
		_, err := mp.Read(buf)
		done <- err
	}()

	// Small delay to ensure the Read goroutine is blocked
	time.Sleep(50 * time.Millisecond)
	mp.Close()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after Close, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: Close should have unblocked Read")
	}
}

func TestMultiPlexerConcurrentInputs(t *testing.T) {
	const n = 10
	readers := make([]io.ReadCloser, n)
	for i := range n {
		readers[i] = io.NopCloser(strings.NewReader("x"))
	}
	mp := NewMultiPlexer(readers...)

	go mp.Listen()

	resultCh := make(chan []byte, 1)
	go func() {
		var mu sync.Mutex
		var result []byte
		buf := make([]byte, 1024)
		for {
			k, err := mp.Read(buf)
			if k > 0 {
				mu.Lock()
				result = append(result, buf[:k]...)
				mu.Unlock()
			}
			if err != nil {
				break
			}
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		if len(result) < n {
			t.Fatalf("expected at least %d bytes, got %d", n, len(result))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading concurrent inputs")
	}
}

func TestMultiPlexerCustomLogger(t *testing.T) {
	// Use a syncWriter-backed buffer so the logger's writes (from the Listen
	// goroutine) and the test's reads are protected by the same mutex.
	var logBuf bytes.Buffer
	sw := NewSyncWriter(&logBuf)
	r := io.NopCloser(strings.NewReader("data"))
	mp := NewMultiPlexer(r)
	mp.Logger = NewLogger(sw, "test")

	listenDone := make(chan struct{})
	go func() {
		mp.Listen()
		close(listenDone)
	}()

	// Drain the pipe until EOF
	buf := make([]byte, 64)
	for {
		n, err := mp.Read(buf)
		_ = n
		if err != nil {
			break
		}
	}

	// Wait for the Listen goroutine to fully exit before reading the log buffer.
	select {
	case <-listenDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Listen to complete")
	}

	// Now it's safe to read logBuf (Listen goroutine has exited).
	sw.mu.Lock()
	n := logBuf.Len()
	sw.mu.Unlock()
	if n == 0 {
		t.Fatal("expected log output with custom logger")
	}
}

func TestMultiPlexerErrHandlerCalled(t *testing.T) {
	// Use a reader that returns an error (not EOF) to trigger ErrHandler.
	// io.Pipe: close the writer with an error.
	pr, pw := io.Pipe()
	mp := NewMultiPlexer(pr)

	errCh := make(chan error, 1)
	mp.ErrHandler = func(err error) {
		select {
		case errCh <- err:
		default:
		}
	}

	go mp.Listen()

	// Trigger a non-EOF error on the reader
	pw.CloseWithError(io.ErrUnexpectedEOF)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error in ErrHandler")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ErrHandler to be called")
	}
}
