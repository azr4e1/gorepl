//go:build windows

package internals

import "errors"

var errNotSupported = errors.New("named pipes not supported on windows")

const NPipeName = "named_pipe"

type TempNPipe struct{ Address string }

func MkTempFifo(bool) (*TempNPipe, error)      { return nil, errNotSupported }
func NewTempFifo() (*TempNPipe, error)         { return nil, errNotSupported }
func NewFifo(string) (*TempNPipe, error)       { return nil, errNotSupported }
func (t *TempNPipe) Read([]byte) (int, error)  { return 0, errNotSupported }
func (t *TempNPipe) Write([]byte) (int, error) { return 0, errNotSupported }
func (t *TempNPipe) Close() error              { return nil }
func (t *TempNPipe) CleanUp() error            { return nil }
func GenerateNPipePath() (string, error)       { return "", errNotSupported }
