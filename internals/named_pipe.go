package internals

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path"
)

type TempNPipe struct {
	tempDir string
	name    string
	fd      *os.File
}

func MkTempFifo(name string) (*TempNPipe, error) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-*", name))
	if err != nil {
		return nil, err
	}
	npipePath := path.Join(tempDir, fmt.Sprintf("%s-namedpipe", name))
	err = unix.Mkfifo(npipePath, 0666)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	return &TempNPipe{
		tempDir: tempDir,
		name:    npipePath,
		fd:      fd,
	}, nil
}

func (unp *TempNPipe) Read(p []byte) (int, error) {
	return unp.fd.Read(p)
}

func (unp *TempNPipe) Write(p []byte) (int, error) {
	return unp.fd.Write(p)
}

func (unp *TempNPipe) Close() error {
	err := unp.fd.Close()
	if err != nil {
		return err
	}
	err = os.RemoveAll(unp.tempDir)
	return err
}
