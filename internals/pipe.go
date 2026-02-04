package internals

import (
	"io"
	"log"
)

type MultiPlexer struct {
	// All FDs the pipe must read from
	Inputs []io.Reader
	// Channel that collects data from Readers to buffer
	bufChan chan []byte
	// Channel to respond to read request: sends n bytes of buffer
	writeChan chan []byte
	// Buffered data
	buffer        []byte
	dataRequested int
}

func NewMultiPlexer(inputs []io.Reader) *MultiPlexer {
	funnelReader := &MultiPlexer{
		Inputs:    inputs,
		bufChan:   make(chan []byte),
		writeChan: make(chan []byte),
		buffer:    []byte{},
	}

	go funnelReader.listen()

	return funnelReader
}

func readInput(fd io.Reader, xcomms chan []byte) error {
	buf := make([]byte, BufSize)
	for {
		n, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n > 0 {
			xcomms <- buf[:n]
		}
	}
}

func (mp *MultiPlexer) listen() {
	for _, i := range mp.Inputs {
		input := i
		go func() {
			err := readInput(input, mp.bufChan)
			if err != nil {
				log.Println(err)
			}
		}()
	}

	for {
		data := <-mp.bufChan
		mp.buffer = append(mp.buffer, data...)
		if n := mp.dataRequested; n > 0 {
			readN := min(n, len(mp.buffer))
			data := mp.buffer[:readN]
			mp.buffer = mp.buffer[readN:]
			mp.writeChan <- data
			mp.dataRequested = 0
		}
	}
}

func (mp *MultiPlexer) Read(buf []byte) (int, error) {
	bufLen := len(buf)
	mp.dataRequested = bufLen
	data := <-mp.writeChan
	n := len(data)

	for i := 0; i < n; i++ {
		buf[i] = data[i]
	}

	return n, nil
}
