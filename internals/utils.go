package internals

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"sync"
)

var DiscardLogger = NewLogger(io.Discard, "")
var DefaultErrHandler = func(err error) {}

type ErrHandler func(error)

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

func NewSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{
		mu: sync.Mutex{},
		w:  w,
	}
}

func NewLogger(w io.Writer, name string) *log.Logger {
	sw := NewSyncWriter(w)
	// prefix construction
	sub := "%s"
	if name != "" {
		sub += ":"
	}
	sub += " "
	logger := log.New(sw, fmt.Sprintf(sub, name), log.Ltime|log.Ldate|log.Lmsgprefix)

	return logger
}

type ReaderWithEcho struct {
	reader     io.ReadCloser
	echoWriter io.Writer
}

func NewReaderWithEcho(reader io.ReadCloser, echo io.Writer) *ReaderWithEcho {
	return &ReaderWithEcho{
		reader:     reader,
		echoWriter: echo,
	}
}

func (re *ReaderWithEcho) Read(p []byte) (int, error) {
	n, err := re.reader.Read(p)
	if err != nil {
		return n, err
	}
	_, err = re.echoWriter.Write(p[:n])

	return n, err
}

func (re *ReaderWithEcho) Close() error {
	return re.reader.Close()
}

// Source - https://stackoverflow.com/a/10510783
// Posted by Mostafa, modified by community. See post 'Timeline' for change history
// Retrieved 2026-02-09, License - CC BY-SA 4.0
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
