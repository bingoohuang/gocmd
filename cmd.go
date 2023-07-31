// Package gocmd runs external commands with concurrent access to output and
// status. It wraps the Go standard library os/exec.Command to correctly handle
// reading output (STDOUT and STDERR) while a command is running and killing a
// command. All operations are safe to call from multiple goroutines.
package gocmd

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
	stderrWriter io.Writer
	StdoutWriter io.Writer
	Cmd          *exec.Cmd
	Dir          string
	Command      string
	Args         []string
	WorkingDir   string
	Env          []string

	// StdoutBuf and StdoutBuf retrieve the output after the command was Executed
	StdoutBuf   bytes.Buffer
	CombinedBuf bytes.Buffer
	StderrBuf   bytes.Buffer
	Timeout     time.Duration
	exitCode    int

	Executed bool
}

// EnvVars represents a map where the key is the name of the Env variable
// and the value is the value of the variable
//
// Example:
//
//	Env := map[string]string{"ENV": "VALUE"}
type EnvVars map[string]string

// New creates a new command
// You can add option with variadic option argument
// Default timeout is set to 30 minutes
//
// Example:
//
//	     c := gocmd.New("echo hello", function (c *Cmd) {
//			    c.WorkingDir = "/tmp"
//	     })
//	     c.Run(context.TODO())
//
// or you can use existing options functions
//
//	c := gocmd.New("echo hello", gocmd.WithStdStreams)
//	c.Run(context.TODO())
func New(cmd string, options ...func(*Cmd)) *Cmd {
	c := &Cmd{
		Command: cmd,
		Timeout: 1 * time.Minute,
	}
	c.Env = append(c.Env, os.Environ()...)
	c.Cmd = createBaseCommand(c)
	c.StdoutWriter = io.MultiWriter(&c.StdoutBuf, &c.CombinedBuf)
	c.stderrWriter = io.MultiWriter(&c.StderrBuf, &c.CombinedBuf)

	for _, o := range options {
		o(c)
	}

	return c
}

// Run directly runs a new command
func Run(cmd string, options ...func(*Cmd)) (string, error) {
	c := New(cmd, options...)
	if err := c.Run(context.Background()); err != nil {
		return "", err
	}

	if stderr := c.Stderr(); stderr != "" {
		return "", errors.New(stderr)
	}

	return c.Stdout(), nil
}

// WithCmd allows the OS specific generated baseCommand
// to be overridden by an *os/exec.Cmd.
//
// Example:
//
//	c := gocmd.New("", gocmd.WithCmd(exec.Cmd("echo", "hello")),
//	)
//	c.Run(context.TODO())
func WithCmd(cmd *exec.Cmd) func(c *Cmd) {
	return func(c *Cmd) {
		c.Cmd = cmd
		if c.Command != "" {
			c.Cmd.Args = append(c.Cmd.Args, c.Command)
		}
	}
}

// WithStdStreams is used as an option by the New constructor function and writes the output streams
// to StderrBuf and StdoutBuf of the operating system
//
// Example:
//
//	c := gocmd.New("echo hello", gocmd.WithStdStreams())
//	c.Run(context.TODO())
func WithStdStreams() func(c *Cmd) {
	return func(c *Cmd) {
		c.StdoutWriter = io.MultiWriter(os.Stdout, &c.StdoutBuf, &c.CombinedBuf)
		c.stderrWriter = io.MultiWriter(os.Stderr, &c.StderrBuf, &c.CombinedBuf)
	}
}

// WithStdout allows to add custom writers to StdoutBuf
func WithStdout(writers ...io.Writer) func(c *Cmd) {
	return func(c *Cmd) {
		var allWriters []io.Writer
		allWriters = append(allWriters, &c.StdoutBuf, &c.CombinedBuf)
		allWriters = append(allWriters, writers...)
		c.StdoutWriter = io.MultiWriter(allWriters...)
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
//	gocmd.New("sleep 10;", gocmd.WithTimeout(500))
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
		c.Env = nil
	}
}

// AddEnv adds an environment variable to the command
// If a variable gets passed like ${VAR_NAME} the Env variable will be read out by the current shell
func (c *Cmd) AddEnv(key, value string) {
	value = os.ExpandEnv(value)
	c.Env = append(c.Env, fmt.Sprintf("%s=%s", key, value))
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

// Run runs with Context
func (c *Cmd) Run(ctx context.Context) error {
	cmd := c.Cmd
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.Setpgid = true // // 设置进程组
	cmd.Env = c.Env
	cmd.Dir = c.Dir
	cmd.Stdout = c.StdoutWriter
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

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// gocmd.Process.Kill();
		// Signal the process group (-pid), not just the process, so that the process
		// and all its children are signaled. Else, child procs can keep running and
		// keep the stdout/stderr fd open and cause gocmd.Wait to hang.
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

func (c *Cmd) getExitCode(err error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			c.exitCode = status.ExitStatus()
		}
	}
}
