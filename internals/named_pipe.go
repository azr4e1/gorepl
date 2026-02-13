package internals

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const NPipeName = "named_pipe"

type TempNPipe struct {
	Address string
	fd      *os.File
}

func GenerateNPipePath(tempPath string) string {
	return strings.Join([]string{tempPath, NPipeName}, "_")
}

func MkTempFifo(force bool) (*TempNPipe, error) {
	tempPath, err := GetPathCurDir()
	if err != nil {
		return nil, err
	}

	// check if dir exists; if so, error unless forced
	npipePath := GenerateNPipePath(tempPath)
	if ok, _ := Exists(npipePath); ok {
		if !force {
			return nil, errors.New("pipe already exists at this address")
		}
		err := os.Remove(npipePath)
		if err != nil {
			return nil, err
		}
	}
	err = unix.Mkfifo(npipePath, 0666)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		os.Remove(npipePath)
		return nil, err
	}

	return &TempNPipe{
		Address: npipePath,
		fd:      fd,
	}, nil
}

func NewTempFifo(tempPath string) (*TempNPipe, error) {
	npipePath := GenerateNPipePath(tempPath)
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &TempNPipe{
		Address: npipePath,
		fd:      fd,
	}, nil

}

func (tnp *TempNPipe) Read(p []byte) (int, error) {
	return tnp.fd.Read(p)
}

func (tnp *TempNPipe) Write(p []byte) (int, error) {
	return tnp.fd.Write(p)
}

func (tnp *TempNPipe) Close() error {
	return tnp.fd.Close()
}

func (tnp *TempNPipe) CleanUp() error {
	tnp.Close() // ignore err, best effort
	err := os.Remove(tnp.Address)
	return err
}
