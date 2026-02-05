package internals

import (
	// "bytes"
	"errors"
	"io"
	"log"
	"sync"
)

type MultiPlexer struct {
	// All FDs the pipe must read from
	Inputs  []io.Reader
	Outputs []io.Writer
	// buffer  *bytes.Buffer
	logger *log.Logger
}

func NewMultiPlexer(inputs []io.Reader, output []io.Writer, logger *log.Logger) *MultiPlexer {
	syncOutputs := []io.Writer{}
	for _, w := range output {
		syncOutputs = append(syncOutputs, newSyncWriter(w))
	}
	multiPlexer := &MultiPlexer{
		Inputs:  inputs,
		Outputs: syncOutputs,
		// buffer:  new(bytes.Buffer),
		logger: logger,
	}

	return multiPlexer
}

func (mp *MultiPlexer) Broadcast(p []byte) error {
	errSlice := []error{}
	var err error
	for _, w := range mp.Outputs {
		_, err = w.Write(p)
		errSlice = append(errSlice, err)
	}
	// _, err := mp.buffer.Write(p)
	// errSlice = append(errSlice, err)

	return errors.Join(errSlice...)
}

func (mp *MultiPlexer) pipe(fd io.Reader) error {
	buf := make([]byte, BufSize)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			return err
		}

		if n > 0 {
			mp.logger.Printf("read from input")
			err := mp.Broadcast(buf[:n])
			if err != nil {
				return err
			}
			mp.logger.Printf("written to output")
		}
	}
}

func (mp *MultiPlexer) Listen() {
	var wg sync.WaitGroup
	for _, i := range mp.Inputs {
		input := i
		wg.Add(1)
		go func() {
			err := mp.pipe(input)
			if err != nil {
				mp.logger.Println(err)
			}
			wg.Done()
		}()
	}
	mp.logger.Printf("launched all goroutines")
	mp.logger.Printf("listening")
	wg.Wait()
	// for {
	// 	data := <-mp.bufChan
	// 	mp.buffer = append(mp.buffer, data...)
	// 	if n := mp.dataRequested; n > 0 {
	// 		readN := min(n, len(mp.buffer))
	// 		data := mp.buffer[:readN]
	// 		mp.buffer = mp.buffer[readN:]
	// 		mp.writeChan <- data
	// 		mp.dataRequested = 0
	// 	}
	// 	mp.logger.Printf("multiplexer loop")
	// }

}

// func (mp *MultiPlexer) Read(buf []byte) (int, error) {
// 	bufLen := len(buf)
// 	mp.dataRequested = bufLen
// 	data := <-mp.writeChan
// 	n := len(data)

// 	for i := 0; i < n; i++ {
// 		buf[i] = data[i]
// 	}

// 	mp.logger.Printf("read action")
// 	return n, nil
// }
