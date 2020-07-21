package main

import (
	"io"
	"os/exec"
)

type Runner interface {
	Run(name string, arg ...string) error
	RunWithPipedInput(input string, name string, arg ...string) error
	Stdout() io.Writer
	Stderr() io.Writer
}

type BasicRunner struct {
	Runner
	dir    string
	env    []string
	stdout io.Writer
	stderr io.Writer
}

func NewBasicRunner(dir string, env []string, stdout, stderr io.Writer) *BasicRunner {
	return &BasicRunner{
		dir:    dir,
		env:    env,
		stdout: stdout,
		stderr: stderr,
	}
}

// Run executes the given program.
func (e *BasicRunner) Run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	return cmd.Run()
}

func (e *BasicRunner) RunWithPipedInput(input string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	stdin, _ := cmd.StdinPipe()
	cmd.Start()
	stdin.Write([]byte(input))
	stdin.Close()

	return cmd.Wait()
}

func (e *BasicRunner) Stdout() io.Writer {
	return e.stdout
}

func (e *BasicRunner) Stderr() io.Writer {
	return e.stderr
}
