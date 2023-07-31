//go:build !windows

package gocmd_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bingoohuang/gocmd"
	"github.com/stretchr/testify/assert"
)

func TestCommand_ExecuteStderr1(t *testing.T) {
	c := gocmd.New(">&2 echo hello")
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", c.Stderr())
}

func TestCommand_WithTimeout1(t *testing.T) {
	c := gocmd.New("sleep 0.1;", gocmd.WithTimeout(1*time.Millisecond))
	err := c.Run(context.TODO())

	assert.NotNil(t, err)
	// Sadly a process can not be killed every time :(
	containsMsg := strings.Contains(err.Error(), "timeout, kill") || strings.Contains(err.Error(), "timeout after 1ms")
	assert.True(t, containsMsg)
}

func TestCommand_WithValidTimeout1(t *testing.T) {
	c := gocmd.New("sleep 0.01;", gocmd.WithTimeout(500*time.Millisecond))
	err := c.Run(context.TODO())
	assert.Nil(t, err)
}

func TestCommand_WithWorkingDir(t *testing.T) {
	tempDir := os.TempDir()
	setWorkingDir := func(c *gocmd.Cmd) {
		c.WorkingDir = tempDir
	}

	c := gocmd.New("pwd", setWorkingDir)
	c.Run(context.TODO())

	out := c.Stdout()
	assert.True(t, strings.Contains(out, tempDir[:len(tempDir)-1]))
}

func TestCommand_WithStandardStreams(t *testing.T) {
	tmpFile, _ := os.CreateTemp("/tmp", "stdout_")
	originalStdout := os.Stdout
	os.Stdout = tmpFile

	// Reset os.Stdout to its original value
	defer func() {
		os.Stdout = originalStdout
	}()

	c := gocmd.New("echo hey", gocmd.WithStdStreams())
	c.Run(context.TODO())

	r, err := os.ReadFile(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, "hey\n", string(r))
}

func TestCommand_WithoutTimeout(t *testing.T) {
	c := gocmd.New("sleep 0.001; echo hello", gocmd.WithTimeout(0))
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", c.Stdout())
}

func TestCommand_WithInvalidDir(t *testing.T) {
	c := gocmd.New("echo hello", gocmd.WithWorkingDir("/invalid"))
	err := c.Run(context.TODO())
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), ": no such file or directory"))
}

func TestWithInheritedEnvironment(t *testing.T) {
	os.Setenv("FROM_OS", "is on os")
	os.Setenv("OVERWRITE", "is on os but should be overwritten")
	defer func() {
		os.Unsetenv("FROM_OS")
		os.Unsetenv("OVERWRITE")
	}()

	c := gocmd.New(
		"echo $FROM_OS $OVERWRITE",
		gocmd.WithEnv(map[string]string{"OVERWRITE": "overwritten"}))
	c.Run(context.TODO())

	assertEqualWithLineBreak(t, "is on os overwritten", c.Stdout())
}

func TestWithCustomStderr(t *testing.T) {
	writer := bytes.Buffer{}
	c := gocmd.New(">&2 echo StderrBuf; sleep 0.01; echo StdoutBuf;", gocmd.WithStderr(&writer))
	c.Run(context.TODO())

	assertEqualWithLineBreak(t, "StderrBuf", writer.String())
	assertEqualWithLineBreak(t, "StdoutBuf", c.Stdout())
	assertEqualWithLineBreak(t, "StderrBuf", c.Stderr())
	assertEqualWithLineBreak(t, "StderrBuf\nStdoutBuf", c.Combined())
}

func TestWithCustomStdout(t *testing.T) {
	writer := bytes.Buffer{}
	c := gocmd.New(">&2 echo StderrBuf; sleep 0.01; echo StdoutBuf;", gocmd.WithStdout(&writer))
	c.Run(context.TODO())

	assertEqualWithLineBreak(t, "StdoutBuf", writer.String())
	assertEqualWithLineBreak(t, "StdoutBuf", c.Stdout())
	assertEqualWithLineBreak(t, "StderrBuf", c.Stderr())
	assertEqualWithLineBreak(t, "StderrBuf\nStdoutBuf", c.Combined())
}

func TestWithEnvironmentVariables(t *testing.T) {
	c := gocmd.New("echo $Env", gocmd.WithEnv(map[string]string{"Env": "value"}))
	c.Run(context.TODO())

	assertEqualWithLineBreak(t, "value", c.Stdout())
}

func TestCommand_WithContext(t *testing.T) {
	// ensure legacy timeout is honored
	c := gocmd.New("sleep 3;", gocmd.WithTimeout(1*time.Second))
	err := c.Run(context.TODO())
	assert.NotNil(t, err)
	assert.Equal(t, "timeout after 1s", err.Error())

	// set context timeout to 2 seconds to ensure
	// context takes precedence over timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c = gocmd.New("sleep 3;", gocmd.WithTimeout(1*time.Second))
	err = c.Run(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "context deadline exceeded", err.Error())
}

func TestCommand_WithCustomBaseCommand(t *testing.T) {
	c := gocmd.New(
		"echo $0",
		gocmd.WithBaseCommand(exec.Command("/bin/bash", "-c")),
	)

	err := c.Run(context.TODO())
	assert.Nil(t, err)
	// on darwin we use /bin/sh by default test if we're using bash
	assert.NotEqual(t, "/bin/sh\n", c.Stdout())
	assert.Equal(t, "/bin/bash\n", c.Stdout())
}

func TestCommand_ExecuteStderr(t *testing.T) {
	c := gocmd.New(">&2 echo hello")
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", c.Stderr())
}

func TestCommand_WithTimeout(t *testing.T) {
	c := gocmd.New("sleep 0.5;", gocmd.WithTimeout(5*time.Millisecond))
	err := c.Run(context.TODO())

	assert.NotNil(t, err)
	assert.Equal(t, "timeout after 5ms", err.Error())
}

func TestCommand_WithValidTimeout(t *testing.T) {
	c := gocmd.New("sleep 0.01;", gocmd.WithTimeout(500*time.Millisecond))
	err := c.Run(context.TODO())

	assert.Nil(t, err)
}

// I really don't see the point of mocking this
// as the stdlib does so already. So testing here
// seems redundant. This simple check if we're compliant
// with an api changes
func TestCommand_WithUser(t *testing.T) {
	if runtime.GOOS == "linux" {
		c := gocmd.New("echo hello", gocmd.WithUser(syscall.Credential{Uid: 1111}))
		err := c.Run(context.TODO())
		assert.Equal(t, uint32(1111), c.BaseCommand.SysProcAttr.Credential.Uid)
		assert.Nil(t, err)
	}

	if runtime.GOOS == "darwin" {
		cred := syscall.Credential{}
		c := gocmd.New("echo hello", gocmd.WithUser(cred))
		err := c.Run(context.TODO())
		assert.Error(t, err)
	}
}
