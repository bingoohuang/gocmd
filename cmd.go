// Package cmd runs external commands with concurrent access to output and
// status. It wraps the Go standard library os/exec.Command to correctly handle
// reading output (STDOUT and STDERR) while a command is running and killing a
// command. All operations are safe to call from multiple goroutines.
package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Cmd represents a single command which can be Executed
type Cmd struct {
	Command      string
	env          []string
	Dir          string
	Timeout      time.Duration
	stderrWriter io.Writer
	stdoutWriter io.Writer
	WorkingDir   string
	baseCommand  *exec.Cmd
	Executed     bool
	exitCode     int

	// StdoutBuf and StdoutBuf retrieve the output after the command was Executed
	StdoutBuf   bytes.Buffer
	StderrBuf   bytes.Buffer
	CombinedBuf bytes.Buffer
}

// EnvVars represents a map where the key is the name of the env variable
// and the value is the value of the variable
//
// Example:
//
//	env := map[string]string{"ENV": "VALUE"}
type EnvVars map[string]string

// New creates a new command
// You can add option with variadic option argument
// Default timeout is set to 30 minutes
//
// Example:
//
//	     c := cmd.New("echo hello", function (c *Cmd) {
//			    c.WorkingDir = "/tmp"
//	     })
//	     c.Run()
//
// or you can use existing options functions
//
//	c := cmd.New("echo hello", cmd.WithStdStreams)
//	c.Run()
func New(cmd string, options ...func(*Cmd)) *Cmd {
	c := &Cmd{
		Command: cmd,
		Timeout: 1 * time.Minute,
	}
	c.env = append(c.env, os.Environ()...)
	c.baseCommand = createBaseCommand(c)
	c.stdoutWriter = io.MultiWriter(&c.StdoutBuf, &c.CombinedBuf)
	c.stderrWriter = io.MultiWriter(&c.StderrBuf, &c.CombinedBuf)

	for _, o := range options {
		o(c)
	}

	return c
}

// WithBaseCommand allows the OS specific generated baseCommand
// to be overridden by an *os/exec.Cmd.
//
// Example:
//
//	c := cmd.New(
//	  "echo hello",
//	  cmd.WithBaseCommand(exec.Cmd("/bin/bash", "-c")),
//	)
//	c.Run()
func WithBaseCommand(baseCommand *exec.Cmd) func(c *Cmd) {
	return func(c *Cmd) {
		baseCommand.Args = append(baseCommand.Args, c.Command)
		c.baseCommand = baseCommand
	}
}

// WithStdStreams is used as an option by the New constructor function and writes the output streams
// to StderrBuf and StdoutBuf of the operating system
//
// Example:
//
//	c := cmd.New("echo hello", cmd.WithStdStreams())
//	c.Run()
func WithStdStreams(c *Cmd) func(c *Cmd) {
	return func(c *Cmd) {
		c.stdoutWriter = io.MultiWriter(os.Stdout, &c.StdoutBuf, &c.CombinedBuf)
		c.stderrWriter = io.MultiWriter(os.Stderr, &c.StderrBuf, &c.CombinedBuf)
	}
}

// WithStdout allows to add custom writers to StdoutBuf
func WithStdout(writers ...io.Writer) func(c *Cmd) {
	return func(c *Cmd) {
		var allWriters []io.Writer
		allWriters = append(allWriters, &c.StdoutBuf, &c.CombinedBuf)
		allWriters = append(allWriters, writers...)
		c.stdoutWriter = io.MultiWriter(allWriters...)
	}
}

// WithStderr allows to add custom writers to StderrBuf
func WithStderr(writers ...io.Writer) func(c *Cmd) {
	return func(c *Cmd) {
		var allWriters []io.Writer
		allWriters = append(allWriters, &c.StderrBuf, &c.CombinedBuf)
		allWriters = append(allWriters, writers...)
		c.stderrWriter = io.MultiWriter(allWriters...)
	}
}

// WithTimeout sets the timeout of the command
//
// Example:
//
//	cmd.New("sleep 10;", cmd.WithTimeout(500))
func WithTimeout(t time.Duration) func(c *Cmd) {
	return func(c *Cmd) {
		c.Timeout = t
	}
}

// WithWorkingDir sets the current working directory
func WithWorkingDir(dir string) func(c *Cmd) {
	return func(c *Cmd) {
		c.WorkingDir = dir
	}
}

// WithEnv sets environment variables for the Executed command
func WithEnv(env EnvVars) func(c *Cmd) {
	return func(c *Cmd) {
		for key, value := range env {
			c.AddEnv(key, value)
		}
	}
}

// WithoutEnv clears environment variables for the Executed command
func WithoutEnv() func(c *Cmd) {
	return func(c *Cmd) {
		c.env = nil
	}
}

// AddEnv adds an environment variable to the command
// If a variable gets passed like ${VAR_NAME} the env variable will be read out by the current shell
func (c *Cmd) AddEnv(key, value string) {
	value = os.ExpandEnv(value)
	c.env = append(c.env, fmt.Sprintf("%s=%s", key, value))
}

// Stdout returns the output to StdoutBuf
func (c *Cmd) Stdout() string {
	c.checkExecuted("Stdout")
	return c.StdoutBuf.String()
}

// Stderr returns the output to StderrBuf
func (c *Cmd) Stderr() string {
	c.checkExecuted("Stderr")
	return c.StderrBuf.String()
}

// Combined returns the CombinedBuf output of StderrBuf and StdoutBuf according to their timeline
func (c *Cmd) Combined() string {
	c.checkExecuted("Combined")
	return c.CombinedBuf.String()
}

// ExitCode returns the exit code of the command
func (c *Cmd) ExitCode() int {
	c.checkExecuted("ExitCode")
	return c.exitCode
}

func (c *Cmd) checkExecuted(property string) {
	if c.Executed {
		return
	}

	panic("Can not read " + property + " if command was not Executed.")
}

// RunContext runs Run but with Context
func (c *Cmd) RunContext(ctx context.Context) error {
	cmd := c.baseCommand
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.Setpgid = true // // 设置进程组
	cmd.Env = c.env
	cmd.Dir = c.Dir
	cmd.Stdout = c.stdoutWriter
	cmd.Stderr = c.stderrWriter
	cmd.Dir = c.WorkingDir

	// Respect legacy timer setting only if timeout was set > 0
	// and context does not have a deadline
	_, hasDeadline := ctx.Deadline()
	timeoutCtx := c.Timeout > 0 && !hasDeadline
	if timeoutCtx {
		subCtx, cancel := context.WithTimeout(ctx, c.Timeout)
		defer cancel()
		ctx = subCtx
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		c.Executed = true
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		// cmd.Process.Kill();
		// Signal the process group (-pid), not just the process, so that the process
		// and all its children are signaled. Else, child procs can keep running and
		// keep the stdout/stderr fd open and cause cmd.Wait to hang.
		if err := syscall.Kill(-1*cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("timeout, kill %v: %w", cmd.Process.Pid, err)
		}

		if timeoutCtx {
			return fmt.Errorf("timeout after %v", c.Timeout)
		}
		return ctx.Err()
	case err := <-done:
		c.getExitCode(err)
		return nil
	}
}

// Run executes the command and writes the results into its own instance
// The results can be received with the Stdout(), Stderr() and ExitCode() methods
func (c *Cmd) Run() error {
	return c.RunContext(context.Background())
}

func (c *Cmd) getExitCode(err error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			c.exitCode = status.ExitStatus()
		}
	}
}
