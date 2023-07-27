package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommand_ExecuteStderr(t *testing.T) {
	cmd := New(">&2 echo hello")

	err := cmd.Run()

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", cmd.Stderr())
}

func TestCommand_WithTimeout(t *testing.T) {
	cmd := New("sleep 0.1;", WithTimeout(1*time.Millisecond))

	err := cmd.Run()

	assert.NotNil(t, err)
	// Sadly a process can not be killed every time :(
	containsMsg := strings.Contains(err.Error(), "timeout, kill") || strings.Contains(err.Error(), "timeout after 1ms")
	assert.True(t, containsMsg)
}

func TestCommand_WithValidTimeout(t *testing.T) {
	cmd := New("sleep 0.01;", WithTimeout(500*time.Millisecond))

	err := cmd.Run()

	assert.Nil(t, err)
}

func TestCommand_WithWorkingDir(t *testing.T) {
	setWorkingDir := func(c *Cmd) {
		c.WorkingDir = "/tmp"
	}

	cmd := New("pwd", setWorkingDir)
	cmd.Run()

	assert.Equal(t, "/tmp\n", cmd.Stdout())
}

func TestCommand_WithStandardStreams(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("/tmp", "stdout_")
	originalStdout := os.Stdout
	os.Stdout = tmpFile

	// Reset os.Stdout to its original value
	defer func() {
		os.Stdout = originalStdout
	}()

	cmd := New("echo hey", WithStdStreams)
	cmd.Run()

	r, err := ioutil.ReadFile(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, "hey\n", string(r))
}

func TestCommand_WithoutTimeout(t *testing.T) {
	cmd := New("sleep 0.001; echo hello", WithoutTimeout)

	err := cmd.Run()

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", cmd.Stdout())
}

func TestCommand_WithInvalidDir(t *testing.T) {
	cmd := New("echo hello", WithWorkingDir("/invalid"))
	err := cmd.Run()
	assert.NotNil(t, err)
	assert.Equal(t, "chdir /invalid: no such file or directory", err.Error())
}

func TestWithInheritedEnvironment(t *testing.T) {
	os.Setenv("FROM_OS", "is on os")
	os.Setenv("OVERWRITE", "is on os but should be overwritten")
	defer func() {
		os.Unsetenv("FROM_OS")
		os.Unsetenv("OVERWRITE")
	}()

	c := New(
		"echo $FROM_OS $OVERWRITE",
		WithEnv(map[string]string{"OVERWRITE": "overwritten"}))
	c.Run()

	assertEqualWithLineBreak(t, "is on os overwritten", c.Stdout())
}

func TestWithCustomStderr(t *testing.T) {
	writer := bytes.Buffer{}
	c := New(">&2 echo StderrBuf; sleep 0.01; echo StdoutBuf;", WithStderr(&writer))
	c.Run()

	assertEqualWithLineBreak(t, "StderrBuf", writer.String())
	assertEqualWithLineBreak(t, "StdoutBuf", c.Stdout())
	assertEqualWithLineBreak(t, "StderrBuf", c.Stderr())
	assertEqualWithLineBreak(t, "StderrBuf\nStdoutBuf", c.Combined())
}

func TestWithCustomStdout(t *testing.T) {
	writer := bytes.Buffer{}
	c := New(">&2 echo StderrBuf; sleep 0.01; echo StdoutBuf;", WithStdout(&writer))
	c.Run()

	assertEqualWithLineBreak(t, "StdoutBuf", writer.String())
	assertEqualWithLineBreak(t, "StdoutBuf", c.Stdout())
	assertEqualWithLineBreak(t, "StderrBuf", c.Stderr())
	assertEqualWithLineBreak(t, "StderrBuf\nStdoutBuf", c.Combined())
}

func TestWithEnvironmentVariables(t *testing.T) {
	c := New("echo $env", WithEnv(map[string]string{"env": "value"}))
	c.Run()

	assertEqualWithLineBreak(t, "value", c.Stdout())
}

func TestCommand_WithContext(t *testing.T) {
	// ensure legacy timeout is honored
	cmd := New("sleep 3;", WithTimeout(1*time.Second))
	err := cmd.Run()
	assert.NotNil(t, err)
	assert.Equal(t, "timeout after 1s", err.Error())

	// set context timeout to 2 seconds to ensure
	// context takes precedence over timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd = New("sleep 3;", WithTimeout(1*time.Second))
	err = cmd.RunContext(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "context deadline exceeded", err.Error())
}

func TestCommand_WithCustomBaseCommand(t *testing.T) {
	cmd := New(
		"echo $0",
		WithBaseCommand(exec.Command("/bin/bash", "-c")),
	)

	err := cmd.Run()
	assert.Nil(t, err)
	// on darwin we use /bin/sh by default test if we're using bash
	assert.NotEqual(t, "/bin/sh\n", cmd.Stdout())
	assert.Equal(t, "/bin/bash\n", cmd.Stdout())
}

func TestCommand_WithUser(t *testing.T) {
	cred := syscall.Credential{}
	cmd := New("echo hello", WithUser(cred))
	err := cmd.Run()
	assert.Error(t, err)
}
