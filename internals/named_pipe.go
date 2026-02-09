package internals

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"

	"golang.org/x/sys/unix"
)

const NPipeName = "gorepl_named_pipe"

type TempNPipe struct {
	TempDir string
	Name    string
	fd      *os.File
}

func GetNPipePathCurDir() (string, error) {
	curPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = hash.Write([]byte(curPath))
	if err != nil {
		return "", err
	}
	res := hash.Sum([]byte("_gorepl"))
	hexString := hex.EncodeToString(res)
	tempDir := os.TempDir()

	tempPath := path.Join(tempDir, "gorepl_"+hexString)
	return tempPath, err
}

func MkTempFifo(tempDirPath string, force bool) (*TempNPipe, error) {
	// check if dir exists; if so, error unless forced
	if ok, _ := Exists(tempDirPath); ok {
		if !force {
			return nil, errors.New("pipe already exists at this path")
		}
		err := os.RemoveAll(tempDirPath)
		if err != nil {
			return nil, err
		}
	}
	err := os.Mkdir(tempDirPath, 0777)
	if err != nil {
		return nil, err
	}
	npipePath := path.Join(tempDirPath, NPipeName)
	err = unix.Mkfifo(npipePath, 0666)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		os.RemoveAll(tempDirPath)
		return nil, err
	}

	return &TempNPipe{
		TempDir: tempDirPath,
		Name:    npipePath,
		fd:      fd,
	}, nil
}

func NewTempFifo(tempDirPath string) (*TempNPipe, error) {
	npipePath := path.Join(tempDirPath, NPipeName)
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &TempNPipe{
		TempDir: tempDirPath,
		Name:    npipePath,
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
	err := os.RemoveAll(tnp.TempDir)
	return err
}
