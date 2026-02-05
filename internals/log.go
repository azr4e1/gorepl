package internals

import (
	"fmt"
	"io"
	"log"
	"sync"
)

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
	logger := log.New(sw, fmt.Sprintf("%s: ", name), log.Ltime|log.Ldate|log.Lmsgprefix)

	return logger
}
