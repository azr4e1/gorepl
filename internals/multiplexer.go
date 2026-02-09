package internals

import (
	"errors"
	"io"
	"log"
	"sync"
)

type TransformFunc func(data []byte) []byte

var DefaultTransformFunc = func(data []byte) []byte { return data }

type MultiPlexer struct {
	// All FDs the pipe must read from
	inputs     []io.ReadCloser
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	closeOnce  sync.Once
	Logger     *log.Logger
	Transform  TransformFunc
	ErrHandler ErrHandler
}

func NewMultiPlexer(inputs ...io.ReadCloser) *MultiPlexer {
	pipeReader, pipeWriter := io.Pipe()
	multiPlexer := &MultiPlexer{
		inputs:     inputs,
		pipeReader: pipeReader,
		pipeWriter: pipeWriter,
		Logger:     DiscardLogger,
		Transform:  DefaultTransformFunc,
		ErrHandler: DefaultErrHandler,
	}

	return multiPlexer
}

func (mp *MultiPlexer) pipe(fd io.Reader) error {
	buf := make([]byte, BufSize)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			return err
		}

		if n > 0 {
			mp.Logger.Printf("read from input")
			transformedBuf := mp.Transform(buf[:n])
			_, err := mp.pipeWriter.Write(transformedBuf)
			if err != nil {
				return err
			}
			mp.Logger.Printf("written to output")
		}
	}
}

func (mp *MultiPlexer) Listen() {
	var wg sync.WaitGroup
	for _, i := range mp.inputs {
		input := i
		wg.Go(func() {
			err := mp.pipe(input)
			if err != nil {
				mp.ErrHandler(err)
			}
		})
	}
	mp.Logger.Printf("launched all goroutines")
	mp.Logger.Printf("listening")

	wg.Wait()
	err := mp.Close()
	if err != nil {
		mp.ErrHandler(err)
	}
	mp.Logger.Printf("all streams closed")
}

func (mp *MultiPlexer) Read(p []byte) (int, error) {
	mp.Logger.Print("reading from buffer")
	return mp.pipeReader.Read(p)
}

func (mp *MultiPlexer) Close() error {
	var err error
	// only close once
	mp.closeOnce.Do(func() {
		errList := []error{}
		errList = append(errList, mp.pipeWriter.Close())
		for _, i := range mp.inputs {
			errList = append(errList, i.Close())
		}
		err = errors.Join(errList...)
	})
	return err
}
