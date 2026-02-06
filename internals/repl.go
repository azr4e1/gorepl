// TODO: implement sync writers for buffers; use buffers for buffers - duh, instead of []bytes
package internals

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os/exec"

	"github.com/google/shlex"
)

const BufSize = 4096

type Repl struct {
	Cmd        *exec.Cmd
	ReplStdin  io.WriteCloser
	ReplStdout io.ReadCloser
	ReplStderr io.ReadCloser
	logger     *log.Logger
}

func NewRepl(command string, logger *log.Logger) (*Repl, error) {
	tokens, err := shlex.Split(command)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(context.Background(), tokens[0], tokens[1:]...)

	// pipes
	replStdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	replStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	replStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	repl := &Repl{
		Cmd:        cmd,
		ReplStdin:  replStdin,
		ReplStdout: replStdout,
		ReplStderr: replStderr,
		logger:     logger,
	}

	return repl, nil
}

// GetOutput reads from reader a bufsize amount until there is nothing to read
func (repl *Repl) getOutput(name string, reader io.Reader, writer io.Writer) error {
	buf := make([]byte, BufSize)

	for {
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if n > 0 {
			_, err = writer.Write(buf[:n])
			if err != nil {
				return err
			}
		}
		repl.logger.Printf("%s just processed\n", name)
	}
}

// Read REPL Output and send it to client
func (repl *Repl) SendReplStdOut(clientInput io.Writer) error {
	err := repl.getOutput("stdout", repl.ReplStdout, clientInput)

	return err
}

// Read REPL Error and send it to client
func (repl *Repl) SendReplStdErr(clientInput io.Writer) error {
	err := repl.getOutput("stderr", repl.ReplStderr, clientInput)

	return err
}

// SendToRepl reads from client stdout, and sends lines
// to repl stdin
func (repl *Repl) SendToRepl(clientOutput io.Reader) error {
	scanner := bufio.NewScanner(clientOutput)

	for scanner.Scan() {
		input := scanner.Text()
		io.WriteString(repl.ReplStdin, input+"\n")
		repl.logger.Printf("%s just scanned a line\n", "stdin")
	}
	return scanner.Err()
}

func (repl *Repl) Run(clientOutput io.Reader, clientInput io.Writer, clientErr io.Writer) error {
	// Start REPL
	if err := repl.Cmd.Start(); err != nil {
		return err
	}
	repl.logger.Printf("process started")

	// redirect stdout
	go func() {
		err := repl.SendReplStdOut(clientInput)
		if err != nil {
			if err == io.EOF {
				return
			}
			repl.logger.Print(err)
		}
		repl.logger.Print("stdout closed")
	}()
	repl.logger.Printf("launched stdout redirection")

	// redirect stderr
	go func() {
		err := repl.SendReplStdErr(clientErr)
		if err != nil {
			if err == io.EOF {
				return
			}
			repl.logger.Print(err)
		}
		repl.logger.Print("stderr closed")
	}()
	repl.logger.Printf("launched stderr redirection")

	// redirect stdin
	go func() {
		err := repl.SendToRepl(clientOutput)
		if err != nil {
			repl.logger.Print(err)
			return
		}

		err = repl.Cmd.Cancel()
		if err != nil {
			repl.logger.Print(err)
		}
		repl.logger.Print("stdin closed")
	}()
	repl.logger.Printf("launched stdin redirection")

	if err := repl.Cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			repl.logger.Print("process interrupted")
			return nil
		}
		return err
	}
	repl.logger.Print("process terminated")

	return nil
}
