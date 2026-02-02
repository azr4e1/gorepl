package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
)

func main() {
	sig := make(chan os.Signal, 1)
	done := make(chan bool)
	signal.Notify(sig, os.Interrupt, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "python", "-i")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go func() {
		<-sig

		cancel()

		done <- true
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				fmt.Fprint(os.Stdout, string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				fmt.Fprint(os.Stdout, string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")
	for scanner.Scan() {
		select {
		case <-done:
			fmt.Println("Donzo!")
			os.Exit(1)
		default:
			input := scanner.Text()
			_, err := io.WriteString(stdin, input+"\n")
			if err != nil {
				panic(err)
			}
		}
		fmt.Print("> ")
	}
}
