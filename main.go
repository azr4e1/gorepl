package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
)

const BufSize = 4096

// GetOutput reads from reader a bufsize amount until there is nothing to read
func GetOutput(reader io.ReadCloser, writer io.WriteCloser, bufSize int) error {
	buf := make([]byte, 4096)

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
	}
}

func WriteInput(reader io.ReadCloser, next chan string) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		input := scanner.Text()
		next <- input
	}
	return scanner.Err()
}

func ProcessExit(cmd *exec.Cmd, done chan bool) error {
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	done <- true
	return nil
}

func main() {
	cmd := exec.Command("ipython")
	replStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	replStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	replStderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)
	nextLine := make(chan string)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err := ProcessExit(cmd, done)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// redirect stdout
	go func() {
		err := GetOutput(replStdout, os.Stdout, BufSize)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}
	}()
	// redirect stderr
	go func() {
		err := GetOutput(replStderr, os.Stderr, BufSize)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}
	}()
	go func() {
		err := WriteInput(os.Stdin, nextLine)
		if err != nil {
			log.Fatal(err)
		}
		// reached EOF, break
		done <- true
	}()

	for {
		select {
		case <-done:
			os.Exit(0)
		case line := <-nextLine:
			io.WriteString(replStdin, line+"\n")
		}
	}
}
